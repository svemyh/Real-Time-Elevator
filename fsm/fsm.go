package fsm

import (
	"elevator/elevio"
	"elevator/requests"
	"elevator/timer"
	"fmt"
	"time"
)


// var outputDevice elevio.ElevOutputDevice // Not necessary on Go only in C?

func init() {
	// Initialize the elevator state.
	elevator = ElevatorState{
		Floor:     -1,
		Dir:       elevio.MD_Stop,
		Behaviour: EB_Idle,
		Config: ElevatorConfig{
			DoorOpenDuration: 3 * time.Second,
		},
	}
	//outputDevice = elevio.GetOutputDevice() // Not necessary on Go only in C?
}

// SetAllLights sets the button lamps for all floors and buttons based on the elevator's request state.
func SetAllLights(e ElevatorState) {
	for floor := 0; floor < elevio.N_Floors; floor++ {
		for btn := 0; btn < elevio.N_Buttons; btn++ {
			elevio.SetButtonLamp(elevio.ButtonType(btn), floor, e.Requests[floor][btn])
		}
	}
}

func FsmOnInitBetweenFloors() {
	//outputDevice.SetMotorDirection(elevio.MD_Down)
	elevio.SetMotorDirection(elevio.MD_Down) // added

	elevator.Dir = elevio.MD_Down
	elevator.Behaviour = EB_Moving
}

func FsmOnRequestButtonPress(btnFloor int, btnType elevio.Button) {
	fmt.Printf("\n\n%s(%d, %s)\n", "FsmOnRequestButtonPress", btnFloor, elevio.ButtonToString(btnType))
	//elevatorPrint(elevator)

	switch elevator.Behaviour {
	case EB_DoorOpen:
		if requests.ShouldClearImmediately(elevator, btnFloor, btnType) {
			timer.TimerStart(elevator.Config.DoorOpenDuration.Seconds())
		} else {
			elevator.Requests[btnFloor][btnType] = true
		}
		break

	case EB_Moving:
		elevator.Requests[btnFloor][btnType] = true
		break

	case EB_Idle:
		elevator.Requests[btnFloor][btnType] = true
		pair := requests.ChooseDirection(elevator)
		elevator.Dir = pair.Dirn
		elevator.Behaviour = pair.Behaviour
		switch pair.Behaviour {
		case EB_DoorOpen:
			outputDevice.DoorLight(1)
			timer.TimerStart(elevator.Config.DoorOpenDuration.Seconds())
			elevator = RequestsClearAtCurrentFloor(elevator)
			break

		case EB_Moving:
			outputDevice.MotorDirection(elevator.Dir)
			break

		case EB_Idle:
			break
		}
		break
	}

	SetAllLights(elevator)

	fmt.Println("\nNew state:")
	//elevatorPrint(elevator)
}

func FsmOnFloorArrival(newFloor int) {
	fmt.Printf("\nArrived at floor %d\n", newFloor)
	elevator.Floor = newFloor
	//outputDevice.SetFloorIndicator(newFloor)
	// Additional logic based on elevator state...
}

func FsmOnDoorTimeout() {
	fmt.Println("\nDoor timeout")
	// Additional logic based on elevator state...
}
