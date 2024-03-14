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
var outputDevice elevio.ElevOutputDevice // Is this used anywhere?

func init() {
	// Initialize the elevator state.
	// elevator = elevator.Elevator{
	// 	Floor:      -1,
	// 	Dirn:       elevio.MD_Stop
	// 	Behaviour:  EB_Idle,
	// 	Config:     ElevatorConfig{
	// 		DoorOpenDuration: 3 * time.Second,
	// 	},
	// }
	elevatorState = elevator.ElevatorInit()

	//outputDevice = elevio.GetOutputDevice() // Not necessary on Go only in C?
}

// SetAllLights sets the button lamps for all floors and buttons based on the elevator's request state.
func SetAllCabLights() {
	for floor := 0; floor < elevio.N_Floors; floor++ {
		elevio.SetButtonLamp(elevio.ButtonType(elevio.BT_Cab), floor, elevatorState.Requests[floor][elevio.BT_Cab])
	}
}

func FsmOnInitBetweenFloors() {
	elevio.SetMotorDirection(elevio.D_Down)
	elevatorState.Dirn = elevio.D_Down
	elevatorState.Behaviour = elevator.EB_Moving
}

func FsmOnRequestButtonPress(btnFloor int, btnType elevio.Button, FSMHallOrderCompleteCh chan elevio.ButtonEvent) {
	//fmt.Printf("\n\n%s(%d, %s)\n", "FsmOnRequestButtonPress", btnFloor, elevio.ButtonToString(btnType))
	//elevatorPrint(elevator)

	switch elevatorState.Behaviour {
	case elevator.EB_DoorOpen:
		//println("Door Open")
		if requests.ShouldClearImmediately(elevatorState, btnFloor, btnType) {
			timer.TimerStart(5)
			FSMHallOrderCompleteCh <- elevio.ButtonEvent{Floor: btnFloor, Button: elevio.ButtonType(btnType)}
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
			outputDevice.DoorLight = true
			timer.TimerStart(5)
			elevatorState = requests.ClearAtCurrentFloor(elevatorState, FSMHallOrderCompleteCh)

		case elevator.EB_Moving:
			elevio.SetMotorDirection(elevatorState.Dirn)
			//fmt.Println("Elevator state moving dirn", elevatorState.Dirn)

		case elevator.EB_Idle:
			//fmt.Println("EB_Idle")
		}
	}

	SetAllCabLights()

	//fmt.Println("\nNew state:")
}

func FsmOnFloorArrival(newFloor int, FSMHallOrderCompleteCh chan elevio.ButtonEvent) {
	//fmt.Printf("\nArrived at floor %d\n", newFloor)
	elevatorState.Floor = newFloor
	elevio.SetFloorIndicator(elevatorState.Floor)

	switch elevatorState.Behaviour {
	case elevator.EB_Moving:
		if requests.ShouldStop(elevatorState) {
			fmt.Println("Should stop")
			elevio.SetMotorDirection(elevio.D_Stop)
			elevio.SetDoorOpenLamp(true)
			elevatorState = requests.ClearAtCurrentFloor(elevatorState, FSMHallOrderCompleteCh)
			timer.TimerStart(elevatorState.Config.DoorOpenDurationS)
			SetAllCabLights()
			elevatorState.Behaviour = elevator.EB_DoorOpen
		}
	default:
		break
	}
	//fmt.Printf("\nNew state:\n")

}

func FsmOnDoorTimeout(FSMHallOrderCompleteCh chan elevio.ButtonEvent) {
	//fmt.Println("\nDoor timeout")

	switch elevatorState.Behaviour {
	case elevator.EB_DoorOpen:
		elevatorState.Dirn, elevatorState.Behaviour = requests.ChooseDirection(elevatorState)
		switch elevatorState.Behaviour {
		case elevator.EB_DoorOpen:
			//fmt.Println("EB Door Open")
			timer.TimerStart(elevatorState.Config.DoorOpenDurationS)
			elevatorState = requests.ClearAtCurrentFloor(elevatorState, FSMHallOrderCompleteCh)
			SetAllCabLights()
		case elevator.EB_Moving:
			//fmt.Println("EB moving")
			elevio.SetDoorOpenLamp(false)
			elevio.SetMotorDirection(elevatorState.Dirn)

		case elevator.EB_Idle:
			//fmt.Println("EB idle")
			elevio.SetDoorOpenLamp(false)
			elevio.SetMotorDirection(elevio.D_Stop)
		}

	default:
		break
	}
	//fmt.Printf("-------------------------------")
	//fmt.Printf("\nNew state:\n")
}

func FsmRun(device elevio.ElevInputDevice, FSMStateUpdateCh chan hall_request_assigner.ActiveElevator, FSMHallOrderCompleteCh chan elevio.ButtonEvent, FSMAssignedHallRequestsCh chan [elevio.N_Floors][elevio.N_Buttons - 1]bool) {

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
	//time.Sleep(500 * time.Millisecond) // Make sure FsmOnInitBetweenFloors completes before the rest of FsmRun continues

	log.Println("firste elev state to send floor! : ", hall_request_assigner.ActiveElevator{Elevator: elevatorState, MyAddress: network.GetLocalIPv4()}.Elevator.Floor)

	// Polling for new actions/events of the system.
	for {
		select {
		case floor := <-device.FloorSensorCh:
			//fmt.Println("Floor Sensor:", floor)
			if floor != -1 && floor != prev { // Maybe this logic is redundant? TODO: Check it at later time.
				FsmOnFloorArrival(floor, FSMHallOrderCompleteCh)

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
			toBeSentActiveElevatorState.Elevator.Requests[buttonEvent.Floor][buttonEvent.Button] = true
			if elevatorState.Floor != -1 { // Guarantees that FsmOnInitBetweenFloors() is completed before any button-presses are sent.
				FSMStateUpdateCh <- toBeSentActiveElevatorState
			}

			if buttonEvent.Button == elevio.ButtonType(elevio.B_Cab) {
				fmt.Println("I will on this hill")
				FsmOnRequestButtonPress(buttonEvent.Floor, elevio.Button(buttonEvent.Button), FSMHallOrderCompleteCh)
			}

			//FsmOnRequestButtonPress(buttonEvent.Floor, elevio.Button(buttonEvent.Button)) // is called when an order is recieved from primary

		case obstructionSignal := <-device.ObstructionCh:
			fmt.Println("Obstruction Detected", obstructionSignal)
			// TODO: Implement later

		case AssignedHallRequests := <-FSMAssignedHallRequestsCh:

			for i := 0; i < elevio.N_Floors; i++ {
				for j := 0; j < 2; j++ {
					elevatorState.Requests[i][j] = false
					if AssignedHallRequests[i][j] {
						FsmOnRequestButtonPress(i, elevio.Button(j), FSMHallOrderCompleteCh)
						time.Sleep(5 * time.Millisecond)
					}
				}
			}

		default:
			// No action - prevents blocking on channel reads
			time.Sleep(200 * time.Millisecond)
		}
		if timer.TimerTimedOut() { // should be reworked into a channel. TODO: Implement later
			timer.TimerStop()
			FsmOnDoorTimeout(FSMHallOrderCompleteCh)
			// TODO: Implement acceptance tests here?: IF local error (motor failure or other) is detected this infomation can be fed to PrimaryHandler().
		}
	}
}
