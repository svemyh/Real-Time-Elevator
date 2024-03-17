package fsm

import (
	"elevator/elevator"
	"elevator/elevio"
	"elevator/hall_request_assigner"
	"elevator/network"
	"elevator/requests"
	"elevator/timer"
	"fmt"
	"log"
	"time"
)

var (
	elevatorState elevator.Elevator
 	isDoorOpen bool
	doorOpenDurationS int64 = elevator.ElevatorInit().Config.DoorOpenDurationS
	EB_StuckCh bool = false
)

func init() {
	elevatorState = elevator.ElevatorInit()
}

func doorIsOpen(value bool) {
    elevio.SetDoorOpenLamp(value)
    isDoorOpen = value
}

func setAllCabLights() {
	for floor := 0; floor < elevio.N_Floors; floor++ {
		elevio.SetButtonLamp(elevio.ButtonType(elevio.BT_Cab), floor, elevatorState.Requests[floor][elevio.BT_Cab])
	}
}

func handleInitBetweenFloors() {
	elevio.SetMotorDirection(elevio.D_Down)
	elevatorState.Dirn, elevatorState.Behaviour = elevio.D_Down, elevator.EB_Moving
}

func handleRequestButtonPress(btnFloor 					int, 
							  btnType 					elevio.Button, 
							  FSMHallOrderCompleteCh 	chan<- elevio.ButtonEvent, 
							  CabCopyCh 				chan<- [elevio.N_Floors][elevio.N_Buttons]bool,
) {
	switch elevatorState.Behaviour {
	case elevator.EB_DoorOpen:
		//println("Door Open")
		if requests.ShouldClearImmediately(elevatorState, btnFloor, btnType) {
			doorIsOpen(true)
			log.Println("handleRequestButtonPress1 - DoorOpen")
			timer.DoorTimerStart(doorOpenDurationS)
			elevatorState = requests.ClearAtCurrentFloorAndSendUpdate(elevatorState, FSMHallOrderCompleteCh, CabCopyCh)
			if btnType != elevio.B_Cab { // FSMHallOrderCompleteCh only accepts
				FSMHallOrderCompleteCh <- elevio.ButtonEvent{Floor: btnFloor, Button: elevio.ButtonType(btnType)}
			}
		} else {
			elevatorState.Requests[btnFloor][btnType] = true
		}

	case elevator.EB_Moving:
		elevatorState.Requests[btnFloor][btnType] = true

	case elevator.EB_Idle:
		elevatorState.Requests[btnFloor][btnType] = true

		elevatorState.Dirn, elevatorState.Behaviour = requests.ChooseDirection(elevatorState)

		switch elevatorState.Behaviour {
		case elevator.EB_DoorOpen:
			doorIsOpen(true)
			timer.DoorTimerStart(doorOpenDurationS)
			elevatorState = requests.ClearAtCurrentFloorAndSendUpdate(elevatorState, FSMHallOrderCompleteCh, CabCopyCh)
			if btnType != elevio.B_Cab { // FSMHallOrderCompleteCh only accepts
				FSMHallOrderCompleteCh <- elevio.ButtonEvent{Floor: btnFloor, Button: elevio.ButtonType(btnType)}
			}
		case elevator.EB_Moving:
			elevio.SetMotorDirection(elevatorState.Dirn)
			//fmt.Println("Elevator state moving dirn", elevatorState.Dirn)

		case elevator.EB_Idle:
		}
	}
	CabCopyCh <- elevatorState.Requests
	setAllCabLights()
}

func handleFloorArrival(newFloor 				int, 
						FSMHallOrderCompleteCh 	chan<- elevio.ButtonEvent, 
						CabCopyCh 				chan<- [elevio.N_Floors][elevio.N_Buttons]bool,
) {
	//fmt.Printf("\nArrived at floor %d\n", newFloor)
	elevatorState.Floor = newFloor
	elevio.SetFloorIndicator(elevatorState.Floor)

	switch elevatorState.Behaviour {
	case elevator.EB_Moving:
		if requests.ShouldStop(elevatorState) {
			fmt.Println("Should stop")
			elevio.SetMotorDirection(elevio.D_Stop)
			doorIsOpen(true)
			elevatorState = requests.ClearAtCurrentFloorAndSendUpdate(elevatorState, FSMHallOrderCompleteCh, CabCopyCh)
			timer.DoorTimerStart(doorOpenDurationS)
			setAllCabLights()
			elevatorState.Behaviour = elevator.EB_DoorOpen
		}
	default:
		break
	}
}

func handleDoorTimeout(FSMHallOrderCompleteCh 	chan<- elevio.ButtonEvent, 
					   CabCopyCh 				chan<- [elevio.N_Floors][elevio.N_Buttons]bool,
) {
	//fmt.Println("\nDoor timeout")

	switch elevatorState.Behaviour {
	case elevator.EB_DoorOpen:
		elevatorState.Dirn, elevatorState.Behaviour = requests.ChooseDirection(elevatorState)
		switch elevatorState.Behaviour {
		case elevator.EB_DoorOpen:
			doorIsOpen(true)
			timer.DoorTimerStart(doorOpenDurationS)
			elevatorState = requests.ClearAtCurrentFloorAndSendUpdate(elevatorState, FSMHallOrderCompleteCh, CabCopyCh)
			setAllCabLights()
			log.Println("handleDoorTimeout - DoorOpen")
			
		case elevator.EB_Moving:
			doorIsOpen(false)
			elevio.SetMotorDirection(elevatorState.Dirn)

		case elevator.EB_Idle:
			doorIsOpen(false)
			elevio.SetMotorDirection(elevio.D_Stop)
		}

	default:
		break
	}
}

func sendStuckElevatorState(EB_StuckCh bool, FSMStateUpdateCh chan<- hall_request_assigner.ActiveElevator) {
	log.Println("Updated stuck elevator")
	elevatorState.Available = !EB_StuckCh
	toBeSentActiveElevatorState := hall_request_assigner.ActiveElevator{
		Elevator:  elevatorState,
		MyAddress: network.GetLocalIPv4(),
	}
	FSMStateUpdateCh <- toBeSentActiveElevatorState
	//time.Sleep(300 * time.Millisecond)
}

func checkStuckBetweenFloors(EB_StuckCh bool, FSMStateUpdateCh chan<- hall_request_assigner.ActiveElevator) {
	log.Println("-------Starting checkStuck.")

	var lastFloor int = -1
	stuckDuration := time.Duration(elevatorState.Config.DoorOpenDurationS+3) * time.Second

	stuckTimer := time.NewTimer(stuckDuration)
	
	for {
		select {
		case <-stuckTimer.C:
			if elevatorState.Behaviour == elevator.EB_Moving && lastFloor != -1 {
				log.Printf("Elevator is stuck at floor: %d\n", lastFloor)
				EB_StuckCh = true
				sendStuckElevatorState(EB_StuckCh, FSMStateUpdateCh)
				stuckTimer.Reset(stuckDuration)
			}

		case currentFloor := <-elevio.NewElevInputDevice().FloorSensorCh:
			//log.Printf("Last floor: %d, Current floor: %d\n", lastFloor, currentFloor)
			if currentFloor != lastFloor {
				if elevatorState.Behaviour == elevator.EB_Moving {
					lastFloor = currentFloor
					stuckTimer.Reset(stuckDuration)
					log.Printf("Elevator moved to floor: %d\n", currentFloor)
					EB_StuckCh = false
					sendStuckElevatorState(EB_StuckCh, FSMStateUpdateCh)
				}
			}
		}

		if requests.HasRequests(elevatorState) {
			log.Println("Elevator has no more requests. Not checking for stuck condition.")
			stuckTimer.Stop()
			return
		}
		time.Sleep(500 * time.Millisecond)
	}
}

func FSMRun(device 							elevio.ElevInputDevice, 
			FSMStateUpdateCh 				chan<- hall_request_assigner.ActiveElevator, 
			FSMHallOrderCompleteCh 			chan<- elevio.ButtonEvent, 
			FSMAssignedHallRequestsCh 		<-chan [elevio.N_Floors][elevio.N_Buttons - 1]bool, 
			CabCopyCh 						chan<- [elevio.N_Floors][elevio.N_Buttons]bool, 
			InitCabCopy						[elevio.N_Floors]bool, 
) {
	var prev int = -1
	log.Println("is in fsm")

	if f := elevio.GetFloor(); f == -1 {
		handleInitBetweenFloors()
	} else {
		FSMStateUpdateCh <- hall_request_assigner.ActiveElevator{ // Is this the cause of the error "core.exception.AssertError@optimal_hall_requests.d(27): Some elevator is at an invalid floor
			// i.e. hall_request_assigner.exe does not allow for inputs where an elevator is in the floor "-1"/undefined
			Elevator:  elevatorState,
			MyAddress: network.GetLocalIPv4(),
		}
	}

	for i := 0; i < len(InitCabCopy); i++ {
		if InitCabCopy[i] {
			handleRequestButtonPress(i, elevio.Button(elevio.BT_Cab), FSMHallOrderCompleteCh, CabCopyCh)
		}
		CabCopyCh <- elevatorState.Requests
	}
	time.Sleep(500 * time.Millisecond) // Make sure FsmOnInitBetweenFloors completes before the rest of FsmRun continues

	//log.Println("firste elev state to send floor! : ", hall_request_assigner.ActiveElevator{Elevator: elevatorState, MyAddress: network.GetLocalIPv4()}.Elevator.Floor)

	for {
		select {
		case floor := <-device.FloorSensorCh:
			//fmt.Println("Floor Sensor:", floor)
			if floor != -1 && floor != prev { // Maybe this logic is redundant? TODO: Check it at later time.
				handleFloorArrival(floor, FSMHallOrderCompleteCh, CabCopyCh)

				// Send current elevator to master
				toBeSentActiveElevatorState := hall_request_assigner.ActiveElevator{
					Elevator:  elevatorState,
					MyAddress: network.GetLocalIPv4(),
				}
				FSMStateUpdateCh <- toBeSentActiveElevatorState // Remember to maybe implement: If the primary-local-fsm-loop takes too long time - each new ButtonEvent from localfsm needs to be ORed at the primary side instead of the Primary simply reading latest command. Danger for overwriting.
			}
			prev = floor

		case buttonEvent := <-device.RequestButtonCh:
			toBeSentActiveElevatorState := hall_request_assigner.ActiveElevator{
				Elevator:  elevatorState,
				MyAddress: network.GetLocalIPv4(),
			}
			go checkStuckBetweenFloors(EB_StuckCh, FSMStateUpdateCh)

			toBeSentActiveElevatorState.Elevator.Requests[buttonEvent.Floor][buttonEvent.Button] = true
			if elevatorState.Floor != -1 { // Guarantees that FsmOnInitBetweenFloors() is completed before any button-presses are sent.
				FSMStateUpdateCh <- toBeSentActiveElevatorState
			}

			if buttonEvent.Button == elevio.ButtonType(elevio.B_Cab) {
				fmt.Println("Cab press detected")
				handleRequestButtonPress(buttonEvent.Floor, elevio.Button(buttonEvent.Button), FSMHallOrderCompleteCh, CabCopyCh)
			}
			//FsmOnRequestButtonPress(buttonEvent.Floor, elevio.Button(buttonEvent.Button)) // is called when an order is recieved from primary

		case obstructionSignal := <-device.ObstructionCh:
			fmt.Println("Obstruction Detected", obstructionSignal)
			fmt.Println("OUTPUT DEVICE DOORLIGHT: ", isDoorOpen)

			switch elevatorState.Behaviour {
			case elevator.EB_DoorOpen:
				fmt.Println("Obstruction Detected", obstructionSignal)
				for obstructionSignal && isDoorOpen {
					doorIsOpen(true)
					fmt.Println("Obstruction Detected - DoorLight On", isDoorOpen)
					EB_StuckCh = true
					obstructionSignal := <-device.ObstructionCh
					sendStuckElevatorState(EB_StuckCh, FSMStateUpdateCh)
					if !obstructionSignal {
						doorIsOpen(false)
						fmt.Println("Obstruction Cleared - DoorLight Off", isDoorOpen)
						EB_StuckCh = false
						time.Sleep(500 * time.Millisecond)
						sendStuckElevatorState(EB_StuckCh, FSMStateUpdateCh)
						//break // Should be redundant
					}
				}
				//doorIsOpen(false)
				//fmt.Println("Obstruction Cleared - DoorLight Off", isDoorOpen)
				
				//sendStuckElevatorState(EB_StuckCh, FSMStateUpdateCh)

				time.Sleep(500 * time.Millisecond)
			}

		case AssignedHallRequests := <-FSMAssignedHallRequestsCh:
			fmt.Println("AssignedHallRequests := <-FSMAssignedHallRequestsCh", AssignedHallRequests)
			for i := 0; i < elevio.N_Floors; i++ {
				for j := 0; j < elevio.N_Buttons - 1; j++ {
					elevatorState.Requests[i][j] = false
				}
			}
			for i := 0; i < elevio.N_Floors; i++ {
				for j := 0; j < 2; j++ {
					//elevatorState.Requests[i][j] = false
					if AssignedHallRequests[i][j] {
						handleRequestButtonPress(i, elevio.Button(j), FSMHallOrderCompleteCh, CabCopyCh)
						time.Sleep(5 * time.Millisecond)
						go checkStuckBetweenFloors(EB_StuckCh, FSMStateUpdateCh)
					}
				}
			}

		default:
			time.Sleep(100 * time.Millisecond)
		}
		if timer.DoorTimerTimedOut() {
			timer.DoorTimerStop()
			handleDoorTimeout(FSMHallOrderCompleteCh, CabCopyCh)
			// TODO: Implement acceptance tests here?: IF local error (motor failure or other) is detected this infomation can be fed to PrimaryHandler().
		}

	}
}
