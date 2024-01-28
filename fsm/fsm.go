package fsm

import (
	"elevator/elevio"
	"fmt"
	"time"
)

// ElevatorState represents the state of the elevator.
type ElevatorState struct {
	Floor     int
	Dir       elevio.MotorDirection
	Behaviour ElevatorBehaviour
	Requests  [elevio.N_Floors][elevio.N_Buttons]bool
	Config    ElevatorConfig
}

// ElevatorBehaviour represents the behaviour of the elevator.
type ElevatorBehaviour int

const (
	EB_Idle ElevatorBehaviour = iota
	EB_Moving
	EB_DoorOpen
)

// ElevatorConfig holds configuration for the elevator.
type ElevatorConfig struct {
	DoorOpenDuration time.Duration
}

var elevator ElevatorState
var outputDevice elevio.ElevOutputDevice

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
	//outputDevice = elevio.GetOutputDevice() / //?
}

func FsmOnInitBetweenFloors() {
	//outputDevice.SetMotorDirection(elevio.MD_Down)
	elevator.Dir = elevio.MD_Down
	elevator.Behaviour = EB_Moving
}

func FsmOnRequestButtonPress(btnFloor int, btnType elevio.ButtonType) {
	fmt.Printf("\nRequest button pressed at floor %d, button %v\n", btnFloor, btnType)
	elevator.Requests[btnFloor][btnType] = true
	// Additional logic based on elevator state...
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
