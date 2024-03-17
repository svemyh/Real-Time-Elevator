package fsm

import (
	"elevator/elevator"
	"elevator/elevio"
	"elevator/hall_request_assigner"
	"elevator/network"
	"elevator/requests"
	"elevator/timer"
	"fmt"
	"time"
)

var (
	elevatorState     elevator.Elevator
	isDoorOpen        bool
	doorOpenDurationS int64 = elevator.ElevatorInit().Config.DoorOpenDurationS
)

func init() {
	elevatorState = elevator.ElevatorInit()
}

func setAllCabLights() {
	for floor := 0; floor < elevio.N_Floors; floor++ {
		elevio.SetButtonLamp(elevio.ButtonType(elevio.BT_Cab), floor, elevatorState.Requests[floor][elevio.BT_Cab])
	}
}

func handleInitBetweenFloors() {
	elevio.SetMotorDirection(elevio.D_Down)
	elevatorState.Dirn = elevio.D_Down
	elevatorState.Behaviour = elevator.EB_Moving
}

func handleRequestButtonPress(btnFloor int,
	btnType elevio.ButtonType,
	FSMHallOrderCompleteCh chan<- elevio.ButtonEvent,
	CabCopyCh chan<- [elevio.N_Floors][elevio.N_Buttons]bool,
) {

	switch elevatorState.Behaviour {
	case elevator.EB_DoorOpen:
		if requests.ShouldClearImmediately(elevatorState, btnFloor, btnType) {
			timer.TimerStart(doorOpenDurationS)
			if btnType != elevio.BT_Cab {
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
			timer.TimerStart(doorOpenDurationS)
			elevatorState = requests.ClearAtCurrentFloorAndSendCompletedOrder(elevatorState, FSMHallOrderCompleteCh, CabCopyCh)

		case elevator.EB_Moving:
			elevio.SetMotorDirection(elevatorState.Dirn)

		case elevator.EB_Idle:
		}
	}
	CabCopyCh <- elevatorState.Requests
	setAllCabLights()
}

func handleFloorArrival(newFloor int,
	FSMHallOrderCompleteCh chan<- elevio.ButtonEvent,
	CabCopyCh chan<- [elevio.N_Floors][elevio.N_Buttons]bool,
) {
	elevatorState.Floor = newFloor
	elevio.SetFloorIndicator(elevatorState.Floor)

	switch elevatorState.Behaviour {
	case elevator.EB_Moving:
		if requests.ShouldStop(elevatorState) {
			elevio.SetMotorDirection(elevio.D_Stop)
			elevio.SetDoorOpenLamp(true)
			elevatorState = requests.ClearAtCurrentFloorAndSendCompletedOrder(elevatorState, FSMHallOrderCompleteCh, CabCopyCh)
			timer.TimerStart(doorOpenDurationS)
			setAllCabLights()
			elevatorState.Behaviour = elevator.EB_DoorOpen
		}
	default:
		break
	}
	//fmt.Printf("\nNew state:\n")

}

func handleDoorTimeout(FSMHallOrderCompleteCh chan<- elevio.ButtonEvent,
	CabCopyCh chan<- [elevio.N_Floors][elevio.N_Buttons]bool,
) {

	switch elevatorState.Behaviour {
	case elevator.EB_DoorOpen:
		elevatorState.Dirn, elevatorState.Behaviour = requests.ChooseDirection(elevatorState)
		switch elevatorState.Behaviour {
		case elevator.EB_DoorOpen:
			timer.TimerStart(doorOpenDurationS)
			elevatorState = requests.ClearAtCurrentFloorAndSendCompletedOrder(elevatorState, FSMHallOrderCompleteCh, CabCopyCh)
			setAllCabLights()
		case elevator.EB_Moving:
			elevio.SetDoorOpenLamp(false)
			elevio.SetMotorDirection(elevatorState.Dirn)

		case elevator.EB_Idle:
			elevio.SetDoorOpenLamp(false)
			elevio.SetMotorDirection(elevio.D_Stop)
		}

	default:
		break
	}
}

func FsmRun(device 						elevio.ElevInputDevice,
			FSMStateUpdateCh 			chan<- hall_request_assigner.ActiveElevator,
			FSMHallOrderCompleteCh 		chan<- elevio.ButtonEvent,
			FSMAssignedHallRequestsCh 	<-chan [elevio.N_Floors][elevio.N_Buttons - 1]bool,
			CabCopyCh 					chan<- [elevio.N_Floors][elevio.N_Buttons]bool,
			InitCabCopy 				[elevio.N_Floors]bool,
) {
	var prev int = -1

	if f := elevio.GetFloor(); f == -1 {
		handleInitBetweenFloors()
	} else { // Send current elevator to master
		FSMStateUpdateCh <- hall_request_assigner.ActiveElevator{
			Elevator:  elevatorState,
			MyAddress: network.GetLocalIPv4(),
		}
	}

	for i := 0; i < len(InitCabCopy); i++ {
		if InitCabCopy[i] {
			handleRequestButtonPress(i, elevio.ButtonType(elevio.BT_Cab), FSMHallOrderCompleteCh, CabCopyCh)
		}
		CabCopyCh <- elevatorState.Requests
	}

	for {
		select {
		case floor := <-device.FloorSensorCh:
			if floor != -1 && floor != prev { // Maybe this logic is redundant? TODO: Check it at later time.
				handleFloorArrival(floor, FSMHallOrderCompleteCh, CabCopyCh)

				toBeSentActiveElevatorState := hall_request_assigner.ActiveElevator{
					Elevator:  elevatorState,
					MyAddress: network.GetLocalIPv4(),
				}
				FSMStateUpdateCh <- toBeSentActiveElevatorState
			}
			prev = floor

		case buttonEvent := <-device.RequestButtonCh:
			toBeSentActiveElevatorState := hall_request_assigner.ActiveElevator{
				Elevator:  elevatorState,
				MyAddress: network.GetLocalIPv4(),
			}
			toBeSentActiveElevatorState.Elevator.Requests[buttonEvent.Floor][buttonEvent.Button] = true
			if elevatorState.Floor != -1 { // Guarantees that FsmOnInitBetweenFloors() is completed before any button-presses are sent.
				FSMStateUpdateCh <- toBeSentActiveElevatorState
			}

			if buttonEvent.Button == elevio.ButtonType(elevio.BT_Cab) {
				fmt.Println("Cab press detected")
				handleRequestButtonPress(buttonEvent.Floor, elevio.ButtonType(buttonEvent.Button), FSMHallOrderCompleteCh, CabCopyCh)
			}

		case obstructionSignal := <-device.ObstructionCh:
			fmt.Println("Obstruction Detected", obstructionSignal)

		case AssignedHallRequests := <-FSMAssignedHallRequestsCh:
			fmt.Println("AssignedHallRequests := <-FSMAssignedHallRequestsCh", AssignedHallRequests)
			for i := 0; i < elevio.N_Floors; i++ {
				for j := 0; j < 2; j++ {
					elevatorState.Requests[i][j] = false
				}
			}
			for i := 0; i < elevio.N_Floors; i++ {
				for j := 0; j < 2; j++ {
					//elevatorState.Requests[i][j] = false
					if AssignedHallRequests[i][j] {
						handleRequestButtonPress(i, elevio.ButtonType(j), FSMHallOrderCompleteCh, CabCopyCh)
						time.Sleep(5 * time.Millisecond)
					}
				}
			}

		default:
			// No action - prevents blocking on channel reads
			time.Sleep(200 * time.Millisecond)
		}
		if timer.TimerTimedOut() {
			timer.TimerStop()
			handleDoorTimeout(FSMHallOrderCompleteCh, CabCopyCh)
		}
	}
}
