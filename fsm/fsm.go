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

// var outputDevice elevio.ElevOutputDevice // Not necessary on Go only in C?

var elevatorState elevator.Elevator
var isDoorOpen bool

func init() {
	elevatorState = elevator.ElevatorInit()
}

// SetAllLights sets the button lamps for all floors and buttons based on the elevator's request state.
func SetAllCabLights() {
	for floor := 0; floor < elevio.N_Floors; floor++ {
		elevio.SetButtonLamp(elevio.ButtonType(elevio.BT_Cab), floor, elevatorState.Requests[floor][elevio.BT_Cab])
	}
}

func FsmOnInitBetweenFloors() {
	elevio.SetMotorDirection(elevio.D_Down)
	elevatorState.Dirn, elevatorState.Behaviour = elevio.D_Down, elevator.EB_Moving
}

func FsmOnRequestButtonPress(btnFloor int, btnType elevio.Button, FSMHallOrderCompleteCh chan elevio.ButtonEvent, CabCopyCh chan [elevio.N_Floors][elevio.N_Buttons]bool) {
	//fmt.Printf("\n\n%s(%d, %s)\n", "FsmOnRequestButtonPress", btnFloor, elevio.ButtonToString(btnType))
	//elevatorPrint(elevator)

	switch elevatorState.Behaviour {
	case elevator.EB_DoorOpen:
		//println("Door Open")
		if requests.ShouldClearImmediately(elevatorState, btnFloor, btnType) {
			timer.TimerStart(5)
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
			elevio.SetDoorOpenLamp(true)
			isDoorOpen = true
			timer.TimerStart(5)
			elevatorState = requests.ClearAtCurrentFloor(elevatorState, FSMHallOrderCompleteCh, CabCopyCh)

		case elevator.EB_Moving:
			elevio.SetMotorDirection(elevatorState.Dirn)
			//fmt.Println("Elevator state moving dirn", elevatorState.Dirn)

		case elevator.EB_Idle:
			//fmt.Println("EB_Idle")
		}
	}
	CabCopyCh <- elevatorState.Requests
	SetAllCabLights()
	//fmt.Println("\nNew state:")
}

func FsmOnFloorArrival(newFloor int, FSMHallOrderCompleteCh chan elevio.ButtonEvent, CabCopyCh chan [elevio.N_Floors][elevio.N_Buttons]bool) {
	//fmt.Printf("\nArrived at floor %d\n", newFloor)
	elevatorState.Floor = newFloor
	elevio.SetFloorIndicator(elevatorState.Floor)

	switch elevatorState.Behaviour {
	case elevator.EB_Moving:
		if requests.ShouldStop(elevatorState) {
			fmt.Println("Should stop")
			elevio.SetMotorDirection(elevio.D_Stop)
			elevio.SetDoorOpenLamp(true)
			isDoorOpen = true
			elevatorState = requests.ClearAtCurrentFloor(elevatorState, FSMHallOrderCompleteCh, CabCopyCh)
			timer.TimerStart(elevatorState.Config.DoorOpenDurationS)
			SetAllCabLights()
			elevatorState.Behaviour = elevator.EB_DoorOpen
		}
	default:
		break
	}
	//fmt.Printf("\nNew state:\n")

}

func FsmOnDoorTimeout(FSMHallOrderCompleteCh chan elevio.ButtonEvent, CabCopyCh chan [elevio.N_Floors][elevio.N_Buttons]bool) {
	//fmt.Println("\nDoor timeout")

	switch elevatorState.Behaviour {
	case elevator.EB_DoorOpen:
		elevatorState.Dirn, elevatorState.Behaviour = requests.ChooseDirection(elevatorState)
		switch elevatorState.Behaviour {
		case elevator.EB_DoorOpen:
			//fmt.Println("EB Door Open")
			timer.TimerStart(elevatorState.Config.DoorOpenDurationS)
			elevatorState = requests.ClearAtCurrentFloor(elevatorState, FSMHallOrderCompleteCh, CabCopyCh)
			SetAllCabLights()
		case elevator.EB_Moving:
			//fmt.Println("EB moving")
			elevio.SetDoorOpenLamp(false)
			isDoorOpen = false
			elevio.SetMotorDirection(elevatorState.Dirn)

		case elevator.EB_Idle:
			//fmt.Println("EB idle")
			elevio.SetDoorOpenLamp(false)
			isDoorOpen = false
			elevio.SetMotorDirection(elevio.D_Stop)
		}

	default:
		break
	}
	//fmt.Printf("-------------------------------")
	//fmt.Printf("\nNew state:\n")
}

func checkStuck(EB_StuckCh chan bool) {
	log.Println("Starting checkStuck.")

	var lastFloor int = -1
	stuckDuration := time.Duration(elevatorState.Config.DoorOpenDurationS + 3) * time.Second

    stuckTimer := time.NewTimer(stuckDuration)
    stuckTimer.Stop() 

	for {
		select {
		case <-stuckTimer.C:
			if elevatorState.Behaviour == elevator.EB_Moving && lastFloor != -1 {
				log.Printf("Elevator is stuck at floor: %d\n", lastFloor)
				EB_StuckCh <- true 
				stuckTimer.Reset(stuckDuration) 
			}
		
		case devicefloor := <-elevio.NewElevInputDevice().FloorSensorCh:
			log.Printf("Last floor: %d, Current floor: %d\n", lastFloor, devicefloor)
			if devicefloor != lastFloor {
				if elevatorState.Behaviour == elevator.EB_Moving {
					lastFloor = devicefloor
					stuckTimer.Reset(stuckDuration)
					log.Printf("Elevator moved to floor: %d\n", devicefloor)
					EB_StuckCh <- false 
				}
			}
		}
		if requests.IsRequestsEmpty(elevatorState) {
			log.Println("Elevator has no more requests. Not checking for stuck condition.")
			return
		}

		time.Sleep(500 * time.Millisecond)
	}
}

/*
func checkStuck(EB_StuckCh chan bool) {
	log.Println("Starting checkStuck.")

	var lastFloor int = -1
	var lastTime time.Time = time.Now()
	stuckDuration := 1 * time.Second

	for {
		devicefloor := <-elevio.NewElevInputDevice().FloorSensorCh

		log.Printf("Last floor: %d, Current floor: %d\n", lastFloor, devicefloor)

		if elevatorState.Behaviour == elevator.EB_Moving {
			log.Println(time.Since(lastTime) >= stuckDuration && lastFloor != -1)
			if time.Since(lastTime) >= stuckDuration && lastFloor != -1 {
				log.Printf("Elevator is stuck at floor: %d\n", lastFloor)
				EB_StuckCh <- true
				// Reset the timer to avoid spamming
				lastTime = time.Now()
			} else if devicefloor != lastFloor {
				lastTime = time.Now()
				lastFloor = devicefloor
				EB_StuckCh <- false
				log.Printf("Elevator moved to floor: %d\n", devicefloor)
			}
		} else {
			log.Println("Elevator door is open; not checking for stuck condition.")
			return
		}

		// Sleep to reduce CPU usage since this is a polling approach
		time.Sleep(500 * time.Millisecond)
	}
}
*/


func FsmRun(device elevio.ElevInputDevice, FSMStateUpdateCh chan hall_request_assigner.ActiveElevator, FSMHallOrderCompleteCh chan elevio.ButtonEvent, FSMAssignedHallRequestsCh chan [elevio.N_Floors][elevio.N_Buttons - 1]bool, CabCopyCh chan [elevio.N_Floors][elevio.N_Buttons]bool, InitCabCopy [elevio.N_Floors]bool, EB_StuckCh chan bool) {

	//elevatorState = elevator.ElevatorInit()
	var prev int = -1
	log.Println("is in fsm")

	if f := elevio.GetFloor(); f == -1 {
		FsmOnInitBetweenFloors()
	} else {
		FSMStateUpdateCh <- hall_request_assigner.ActiveElevator{ // Is this the cause of the error "core.exception.AssertError@optimal_hall_requests.d(27): Some elevator is at an invalid floor
			// i.e. hall_request_assigner.exe does not allow for inputs where an elevator is in the floor "-1"/undefined
			Elevator:  elevatorState,
			MyAddress: network.GetLocalIPv4(),
		}
	}

	for i := 0; i < len(InitCabCopy); i++ {
		if InitCabCopy[i] {
			FsmOnRequestButtonPress(i, elevio.Button(elevio.BT_Cab), FSMHallOrderCompleteCh, CabCopyCh)
		}
		CabCopyCh <- elevatorState.Requests
	}
	//time.Sleep(500 * time.Millisecond) // Make sure FsmOnInitBetweenFloors completes before the rest of FsmRun continues

	log.Println("firste elev state to send floor! : ", hall_request_assigner.ActiveElevator{Elevator: elevatorState, MyAddress: network.GetLocalIPv4()}.Elevator.Floor)

	// Polling for new actions/events of the system.

	for {
		select {
		case floor := <-device.FloorSensorCh:
			//fmt.Println("Floor Sensor:", floor)
			if floor != -1 && floor != prev { // Maybe this logic is redundant? TODO: Check it at later time.
				FsmOnFloorArrival(floor, FSMHallOrderCompleteCh, CabCopyCh)

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
			go checkStuck(EB_StuckCh)

			toBeSentActiveElevatorState.Elevator.Requests[buttonEvent.Floor][buttonEvent.Button] = true
			if elevatorState.Floor != -1 { // Guarantees that FsmOnInitBetweenFloors() is completed before any button-presses are sent.
				FSMStateUpdateCh <- toBeSentActiveElevatorState
			}

			if buttonEvent.Button == elevio.ButtonType(elevio.B_Cab) {
				fmt.Println("Cab press detected")
				FsmOnRequestButtonPress(buttonEvent.Floor, elevio.Button(buttonEvent.Button), FSMHallOrderCompleteCh, CabCopyCh)
			}
			//FsmOnRequestButtonPress(buttonEvent.Floor, elevio.Button(buttonEvent.Button)) // is called when an order is recieved from primary

		case obstructionSignal := <-device.ObstructionCh:
			fmt.Println("Obstruction Detected", obstructionSignal)
			fmt.Println("OUTPUT DEVICE DOORLIGHT: ", isDoorOpen)

			switch elevatorState.Behaviour {
			case elevator.EB_DoorOpen:
				fmt.Println("Obstruction Detected", obstructionSignal)
				for obstructionSignal && isDoorOpen {
					elevio.SetDoorOpenLamp(true)
					isDoorOpen = true
					fmt.Println("Obstruction Detected - DoorLight On", isDoorOpen)
					obstructionSignal := <-device.ObstructionCh
					if !obstructionSignal {
						elevio.SetDoorOpenLamp(false)
						isDoorOpen = false
						fmt.Println("Obstruction Cleared - DoorLight Off", isDoorOpen)
						time.Sleep(500 * time.Millisecond)
						break
					}

				}
				elevio.SetDoorOpenLamp(false)
				isDoorOpen = false
				fmt.Println("Obstruction Cleared - DoorLight Off", isDoorOpen)
				time.Sleep(500 * time.Millisecond)
			}

		case AssignedHallRequests := <-FSMAssignedHallRequestsCh:

			for i := 0; i < elevio.N_Floors; i++ {
				for j := 0; j < 2; j++ {
					elevatorState.Requests[i][j] = false
					if AssignedHallRequests[i][j] {
						FsmOnRequestButtonPress(i, elevio.Button(j), FSMHallOrderCompleteCh, CabCopyCh)
						time.Sleep(100 * time.Millisecond)
					}
				}
			}

		default:
			// No action - prevents blocking on channel reads
			time.Sleep(100 * time.Millisecond)
		}
		if timer.TimerTimedOut() { // should be reworked into a channel. TODO: Implement later
			timer.TimerStop()
			FsmOnDoorTimeout(FSMHallOrderCompleteCh, CabCopyCh)
			// TODO: Implement acceptance tests here?: IF local error (motor failure or other) is detected this infomation can be fed to PrimaryHandler().
		}

	}
}
