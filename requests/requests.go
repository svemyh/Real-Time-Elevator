package requests

import (
	"elevator/elevio"
	"elevator/fsm"
)

func RequestsAbove(e elevio.ElevatorState) bool {
	for f := e.Floor + 1; f < elevio.N_Floors; f++ {
		for btn := range e.Requests[f] {
			if e.Requests[f][btn] {
				return true
			}
		}
	}
	return false
}

func RequestsBelow(e elevio.ElevatorState) bool {
	for f := 0; f < e.Floor; f++ {
		for btn := range e.Requests[f] {
			if e.Requests[f][btn] {
				return true
			}
		}
	}
	return false
}

func RequestsHere(e elevio.ElevatorState) bool {
	for btn := range e.Requests[e.Floor] {
		if e.Requests[e.Floor][btn] {
			return true
		}
	}
	return false
}

func ChooseDirection(e elevio.ElevatorState) (elevio.MotorDirection, elevio.ElevatorBehaviour) {
	switch e.Dir {
	case elevio.MD_Up:
		if RequestsAbove(e) {
			return elevio.MD_Up, fsm.EB_Moving
		} else if RequestsHere(e) {
			return elevio.MD_Down, fsm.EB_DoorOpen
		} else if RequestsBelow(e) {
			return elevio.MD_Down, fsm.EB_Moving
		}
	case elevio.MD_Down:
		if RequestsBelow(e) {
			return elevio.MD_Down, fsm.EB_Moving
		} else if RequestsHere(e) {
			return elevio.MD_Up, fsm.EB_DoorOpen
		} else if RequestsAbove(e) {
			return elevio.MD_Up, fsm.EB_Moving
		}
	}
	return elevio.MD_Stop, fsm.EB_Idle
}

func ShouldStop(e fsm.ElevatorState) bool {
	switch e.Dir {
	case elevio.MD_Down:
		return e.Requests[e.Floor][elevio.B_HallDown] || e.Requests[e.Floor][elevio.B_Cab] || !RequestsBelow(e)
	case elevio.MD_Up:
		return e.Requests[e.Floor][elevio.B_HallUp] || e.Requests[e.Floor][elevio.B_Cab] || !RequestsAbove(e)
	default:
		return true
	}
}

func ShouldClearImmediately(e fsm.ElevatorState, btnFloor int, btnType elevio.ButtonType) bool {
	// Assuming you have a configuration for ClearRequestVariant
	switch e.Config.ClearRequestVariant {
	case ClearAll:
		return e.Floor == btnFloor
	case ClearInDirn:
		return e.Floor == btnFloor && ((e.Dir == elevio.MD_Up && btnType == elevio.B_HallUp) ||
			(e.Dir == elevio.MD_Down && btnType == elevio.B_HallDown) ||
			e.Dir == elevio.MD_Stop || btnType == elevio.B_Cab)
	default:
		return false
	}
}

func ClearAtCurrentFloor(e fsm.ElevatorState) fsm.ElevatorState {
	// Implement logic based on e.Config.ClearRequestVariant
	// Update e.Requests accordingly
	return e
}
