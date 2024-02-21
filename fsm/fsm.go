package fsm

import (
	"elevator/elevator"
	"elevator/elevio"
	"elevator/requests"
	"elevator/timer"
	"fmt"
	//"time"
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
	//outputDevice.SetMotorDirection(elevio.MD_Down)
	elevio.SetMotorDirection(elevio.D_Down) // from MD_Down to D_Down
	//elevio.SetMotorDirection(elevio.MotorDirection(elevio.D_Stop)) //test
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
			timer.TimerStart(5) // it had elevator.Config.DoorOpenDurationS.Seconds() as argument?
		} else {
			elevatorState.Requests[btnFloor][btnType] = true
		}
		

	case elevator.EB_Moving:
		println("Moving")
		elevatorState.Requests[btnFloor][btnType] = true
		

	case elevator.EB_Idle:
		println("Idle")
		elevatorState.Requests[btnFloor][btnType] = true
		//pair := requests.ChooseDirection(elevator)
		//elevator.Dirn = pair.Dirn
		//elevator.Behaviour = pair.Behaviour
		elevatorState.Dirn, elevatorState.Behaviour = requests.ChooseDirection(elevatorState)

		switch elevatorState.Behaviour {
		case elevator.EB_DoorOpen:
			fmt.Println("EB_DoorOpen")
			outputDevice.DoorLight = true
			timer.TimerStart(5) // it had elevator.Config.DoorOpenDurationS.Seconds() as argument?
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
	//elevatorPrint(elevator)
}

func FsmOnFloorArrival(newFloor int) {
	fmt.Printf("\nArrived at floor %d\n", newFloor)
	elevatorState.Floor = newFloor
	//outputDevice.SetFloorIndicator(newFloor)
	// Additional logic based on elevator state... not finished
	elevio.SetFloorIndicator(elevatorState.Floor)

	switch elevatorState.Behaviour {
	case elevator.EB_Moving:
		if requests.ShouldStop(elevatorState) {
			fmt.Println("Should stop")
			elevio.SetMotorDirection(elevio.D_Stop) // from MD_Stop to D_Stop
			elevio.SetDoorOpenLamp(true)
			elevatorState = requests.ClearAtCurrentFloor(elevatorState)
			timer.TimerStart(elevatorState.Config.DoorOpenDurationS)
			SetAllLights() // specific floor?
			elevatorState.Behaviour = elevator.EB_DoorOpen
		}
	default:
		break
	}

	fmt.Printf("\nNew state:\n")

}

func FsmOnDoorTimeout() {
	fmt.Println("\nDoor timeout")
	// Additional logic based on elevator state... finished?
	switch elevatorState.Behaviour {
	case elevator.EB_DoorOpen:
		// pair := requests.ChooseDirection(elevatorState) I don't know why this won't work. Pls fix
		// elevatorState.Dirn = pair.Dirn 	// This part doesn't get used anyways, but wouldn't have worked since ChooseDirection returns motordirection
		// elevatorState.Behaviour = pair.Behaviour

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
			elevio.SetMotorDirection(elevio.D_Stop) // MD_Stop to D_Stop
		}
		
	default:
		break
	}
	fmt.Printf("-------------------------------")
	fmt.Printf("\nNew state:\n")
}
