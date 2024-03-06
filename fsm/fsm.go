package fsm

import (
	"elevator/elevator"
	"elevator/elevio"
	"elevator/requests"
	"elevator/timer"
	"fmt"
	"log"
	"time"
)

// var outputDevice elevio.ElevOutputDevice // Not necessary on Go only in C?

var elevatorState elevator.Elevator
var outputDevice elevio.ElevOutputDevice

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

func FsmOnRequestButtonPress(btnFloor int, btnType elevio.Button) {
	fmt.Printf("\n\n%s(%d, %s)\n", "FsmOnRequestButtonPress", btnFloor, elevio.ButtonToString(btnType))
	//elevatorPrint(elevator)

	switch elevatorState.Behaviour {
	case elevator.EB_DoorOpen:
		println("Door Open")
		if requests.ShouldClearImmediately(elevatorState, btnFloor, btnType) {
			timer.TimerStart(5)
		} else {
			elevatorState.Requests[btnFloor][btnType] = true
		}

	case elevator.EB_Moving:
		println("Moving")
		elevatorState.Requests[btnFloor][btnType] = true

	case elevator.EB_Idle:
		println("Idle")
		elevatorState.Requests[btnFloor][btnType] = true
		elevatorState.Dirn, elevatorState.Behaviour = requests.ChooseDirection(elevatorState)

		switch elevatorState.Behaviour {
		case elevator.EB_DoorOpen:
			fmt.Println("EB_DoorOpen")
			outputDevice.DoorLight = true
			timer.TimerStart(5)
			elevatorState = requests.ClearAtCurrentFloor(elevatorState)

		case elevator.EB_Moving:
			elevio.SetMotorDirection(elevatorState.Dirn)
			fmt.Println("Elevator state moving dirn", elevatorState.Dirn)

		case elevator.EB_Idle:
			fmt.Println("EB_Idle")

		}

	}

	SetAllLights()

	fmt.Println("\nNew state:")
}

func FsmOnFloorArrival(newFloor int) {
	fmt.Printf("\nArrived at floor %d\n", newFloor)
	elevatorState.Floor = newFloor
	elevio.SetFloorIndicator(elevatorState.Floor)

	switch elevatorState.Behaviour {
	case elevator.EB_Moving:
		if requests.ShouldStop(elevatorState) {
			fmt.Println("Should stop")
			elevio.SetMotorDirection(elevio.D_Stop)
			elevio.SetDoorOpenLamp(true)
			elevatorState = requests.ClearAtCurrentFloor(elevatorState)
			timer.TimerStart(elevatorState.Config.DoorOpenDurationS)
			SetAllLights()
			elevatorState.Behaviour = elevator.EB_DoorOpen
		}
	default:
		break
	}

	fmt.Printf("\nNew state:\n")

}

func FsmOnDoorTimeout() {
	fmt.Println("\nDoor timeout")

	switch elevatorState.Behaviour {
	case elevator.EB_DoorOpen:
		elevatorState.Dirn, elevatorState.Behaviour = requests.ChooseDirection(elevatorState)
		switch elevatorState.Behaviour {
		case elevator.EB_DoorOpen:
			fmt.Println("EB Door Open")
			timer.TimerStart(elevatorState.Config.DoorOpenDurationS)
			elevatorState = requests.ClearAtCurrentFloor(elevatorState)
			SetAllLights()
		case elevator.EB_Moving:
			fmt.Println("EB moving")
			elevio.SetDoorOpenLamp(false)
			elevio.SetMotorDirection(elevatorState.Dirn)

		case elevator.EB_Idle:
			fmt.Println("EB idle")
			elevio.SetDoorOpenLamp(false)
			elevio.SetMotorDirection(elevio.D_Stop)
		}

	default:
		break
	}
	fmt.Printf("-------------------------------")
	fmt.Printf("\nNew state:\n")
}

func FsmRun(device elevio.ElevInputDevice) {
	var prev int = -1
	log.Println("is in fsm")

	if f := elevio.GetFloor(); f == -1 {
		FsmOnInitBetweenFloors()
	}

	// Polling for new actions/events of the system.
	for {

		select {
		case floor := <-device.FloorSensorCh:
			fmt.Println("Floor Sensor:", floor)
			if floor != -1 && floor != prev {
				FsmOnFloorArrival(floor)
				//sendToMaster
			}
			prev = floor

		case buttonEvent := <-device.RequestButtonCh:
			fmt.Println("Button Pressed:", buttonEvent)
			FsmOnRequestButtonPress(buttonEvent.Floor, elevio.Button(buttonEvent.Button))

		case obstructionSignal := <-device.ObstructionCh:
			fmt.Println("Obstruction Detected", obstructionSignal)

		default:
			// No action - prevents blocking on channel reads
			time.Sleep(200 * time.Millisecond)
		}
		if timer.TimerTimedOut() { // should be reworked into a channel. TBC
			timer.TimerStop()
			FsmOnDoorTimeout()
		}

	}
}
