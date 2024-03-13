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

func SetAllLights() {
	for floor := 0; floor < elevio.N_Floors; floor++ {
		for btn := 0; btn < elevio.N_Buttons; btn++ {
			elevio.SetButtonLamp(elevio.ButtonType(btn), floor, elevatorState.Requests[floor][btn])
		}
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
		if requests.ShouldClearImmediately(elevatorState, btnFloor, btnType) {
			timer.TimerStart(5)
		} else {
			elevatorState.Requests[btnFloor][btnType] = true
		}

	case elevator.EB_Moving:
		elevatorState.Requests[btnFloor][btnType] = true
		elevio.SetDoorOpenLamp(false)
		isDoorOpen = false

	case elevator.EB_Idle:
		elevatorState.Requests[btnFloor][btnType] = true

		elevatorState.Dirn, elevatorState.Behaviour = requests.ChooseDirection(elevatorState)

		switch elevatorState.Behaviour {
		case elevator.EB_DoorOpen:
			isDoorOpen = true
			timer.TimerStart(5)
			elevatorState = requests.ClearAtCurrentFloor(elevatorState, FSMHallOrderCompleteCh)

		case elevator.EB_Moving:
			elevio.SetMotorDirection(elevatorState.Dirn)

		case elevator.EB_Idle:
		}
	}
	SetAllLights()
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
			isDoorOpen = true
			elevatorState = requests.ClearAtCurrentFloor(elevatorState, FSMHallOrderCompleteCh)
			timer.TimerStart(elevatorState.Config.DoorOpenDurationS)
			SetAllLights()
			elevatorState.Behaviour = elevator.EB_DoorOpen
		}
	default:
		break
	}

}

func FsmOnDoorTimeout(FSMHallOrderCompleteCh chan elevio.ButtonEvent) {

	switch elevatorState.Behaviour {
	case elevator.EB_DoorOpen:
		elevatorState.Dirn, elevatorState.Behaviour = requests.ChooseDirection(elevatorState)
		switch elevatorState.Behaviour {
		case elevator.EB_DoorOpen:
			timer.TimerStart(elevatorState.Config.DoorOpenDurationS)
			elevatorState = requests.ClearAtCurrentFloor(elevatorState, FSMHallOrderCompleteCh)
			SetAllLights()

		case elevator.EB_Moving:
			elevio.SetDoorOpenLamp(false)
			isDoorOpen = false
			elevio.SetMotorDirection(elevatorState.Dirn)

		case elevator.EB_Idle:
			elevio.SetDoorOpenLamp(false)
			isDoorOpen = false
			elevio.SetMotorDirection(elevio.D_Stop)
		}

	default:
		break
	}
}

func FsmRun(device elevio.ElevInputDevice, F network.FSMSystemChannels) {
	//elevatorState = elevator.ElevatorInit()
	var prev int = -1
	log.Println("Fsm is running")

	if f := elevio.GetFloor(); f == -1 {
		FsmOnInitBetweenFloors()
	}
	time.Sleep(500*time.Millisecond) // Make sure FsmOnInitBetweenFloors completes before the rest of FsmRun continues

	log.Println("First elev state to send floor! : ", hall_request_assigner.ActiveElevator{Elevator:  elevatorState, MyAddress: network.GetLocalIPv4()}.Elevator.Floor)
	
	F.FSMStateUpdateCh <- hall_request_assigner.ActiveElevator{ // Is this the cause of the error "core.exception.AssertError@optimal_hall_requests.d(27): Some elevator is at an invalid floor
		// i.e. hall_request_assigner.exe does not allow for inputs where an elevator is in the floor "-1"/undefined
		Elevator:  elevatorState,
		MyAddress: network.GetLocalIPv4(),
	}

	// Polling for new actions/events of the system.
	for {
		select {
		case floor := <-device.FloorSensorCh:
			//fmt.Println("Floor Sensor:", floor)
			if floor != -1 && floor != prev { // Maybe this logic is redundant? TODO: Check it at later time.
				FsmOnFloorArrival(floor, F.FSMHallOrderCompleteCh)

				// Send current elevator to master
				toBeSentActiveElevatorState := hall_request_assigner.ActiveElevator{
					Elevator:  elevatorState,
					MyAddress: network.GetLocalIPv4(),
				}
				F.FSMStateUpdateCh <- toBeSentActiveElevatorState // Remember to maybe implement: If the primary-local-fsm-loop takes too long time - each new ButtonEvent from localfsm needs to be ORed at the primary side instead of the Primary simply reading latest command. Danger for overwriting.
			}
			prev = floor

		case buttonEvent := <-device.RequestButtonCh:
			toBeSentActiveElevatorState := hall_request_assigner.ActiveElevator{
				Elevator:  elevatorState,
				MyAddress: network.GetLocalIPv4(),
			}
			toBeSentActiveElevatorState.Elevator.Requests[buttonEvent.Floor][buttonEvent.Button] = true
			F.FSMStateUpdateCh <- toBeSentActiveElevatorState // Remember to maybe implement: If the primary-local-fsm-loop takes too long time - each new ButtonEvent from localfsm needs to be ORed at the primary side instead of the Primary simply reading latest command. Danger for overwriting.

			if buttonEvent.Button == elevio.ButtonType(elevio.B_Cab) {
				fmt.Println("I will on this hill")
				FsmOnRequestButtonPress(buttonEvent.Floor, elevio.Button(buttonEvent.Button), F.FSMHallOrderCompleteCh)
			}

			//FsmOnRequestButtonPress(buttonEvent.Floor, elevio.Button(buttonEvent.Button)) // is called when an order is recieved from primary

		case obstructionSignal := <-device.ObstructionCh:
		
		fmt.Println("Obstruction Detected", obstructionSignal)
		fmt.Println("OUTPUT DEVICE DOORLIGHT: ", isDoorOpen)


		for obstructionSignal && isDoorOpen{
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
		
		if !obstructionSignal && isDoorOpen {
			elevio.SetDoorOpenLamp(false)
			isDoorOpen = false
			fmt.Println("Obstruction Cleared - DoorLight Off", isDoorOpen)
			time.Sleep(500 * time.Millisecond)
		}
	
		case AssignedHallRequests := <-F.FSMAssignedHallRequestsCh:

			for i := 0; i < elevio.N_Floors; i++ {
				for j := 0; j < 2; j++ {
					elevatorState.Requests[i][j] = false
					if AssignedHallRequests[i][j] {
						FsmOnRequestButtonPress(i, elevio.Button(j), F.FSMHallOrderCompleteCh)
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
			FsmOnDoorTimeout(F.FSMHallOrderCompleteCh)
			// TODO: Implement acceptance tests here?: IF local error (motor failure or other) is detected this infomation can be fed to PrimaryHandler().
		}
	}
}
