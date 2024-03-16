package requests

import (
	"elevator/elevator"
	"elevator/elevio"
	//"elevator/fsm"
)

var elevatorState elevator.Elevator

func RequestsAbove(e elevator.Elevator) bool {
	for f := e.Floor + 1; f < elevio.N_Floors; f++ {
		for btn := range e.Requests[f] {
			if e.Requests[f][btn] {
				return true
			}
		}
	}
	return false
}

func RequestsBelow(e elevator.Elevator) bool {
	for f := 0; f < e.Floor; f++ {
		for btn := range e.Requests[f] {
			if e.Requests[f][btn] {
				return true
			}
		}
	}
	return false
}

func RequestsHere(e elevator.Elevator) bool {
	for btn := range e.Requests[e.Floor] {
		if e.Requests[e.Floor][btn] {
			return true
		}
	}
	return false
}

func IsRequestsEmpty(e elevator.Elevator) bool {
	if RequestsAbove(e) || RequestsBelow(e) || RequestsHere(e) {
		return false
	}

	return true
}

func ChooseDirection(e elevator.Elevator) (elevio.Dirn, elevator.ElevatorBehaviour) { // Elevator doesn't have a motorDirection, but it does have a Dirn
	if RequestsAbove(e) {
		e.Dirn = elevio.D_Up
	} else if RequestsBelow(e) {
		e.Dirn = elevio.D_Down
	} else if RequestsHere(e) {
		e.Dirn = elevio.D_Stop
	}
	switch e.Dirn { // REDUNDANT COPIES
	case elevio.D_Up:
		if RequestsAbove(e) {
			return elevio.D_Up, elevator.EB_Moving
		} else if RequestsHere(e) {
			return elevio.D_Down, elevator.EB_DoorOpen
		} else if RequestsBelow(e) {
			return elevio.D_Down, elevator.EB_Moving
		}
	case elevio.D_Down:
		if RequestsBelow(e) {
			return elevio.D_Down, elevator.EB_Moving
		} else if RequestsHere(e) {
			return elevio.D_Up, elevator.EB_DoorOpen
		} else if RequestsAbove(e) {
			return elevio.D_Up, elevator.EB_Moving
		}
	case elevio.D_Stop:
		if RequestsBelow(e) {
			return elevio.D_Down, elevator.EB_Moving
		} else if RequestsHere(e) {
			return elevio.D_Up, elevator.EB_DoorOpen
		} else if RequestsAbove(e) {
			return elevio.D_Up, elevator.EB_Moving
		}

	}
	return elevio.D_Stop, elevator.EB_Idle
}

func ShouldStop(e elevator.Elevator) bool {
	switch e.Dirn {
	case elevio.D_Down:
		return e.Requests[e.Floor][elevio.B_HallDown] || e.Requests[e.Floor][elevio.B_Cab] || !RequestsBelow(e)
	case elevio.D_Up:
		return e.Requests[e.Floor][elevio.B_HallUp] || e.Requests[e.Floor][elevio.B_Cab] || !RequestsAbove(e)
	default:
		return true
	}
}

func ShouldClearImmediately(e elevator.Elevator, btnFloor int, btnType elevio.Button) bool { // changed to Button from ButtonType
	// Assuming you have a configuration for ClearRequestVariant
	switch e.Config.ClearRequestVariant {
	case elevator.CV_All:
		return e.Floor == btnFloor
	case elevator.CV_InDoorn:
		return e.Floor == btnFloor && ((e.Dirn == elevio.D_Up && btnType == elevio.B_HallUp) ||
			(e.Dirn == elevio.D_Down && btnType == elevio.B_HallDown) ||
			e.Dirn == elevio.D_Stop || btnType == elevio.B_Cab)
	default:
		return false
	}
}

func ClearAtCurrentFloor(e elevator.Elevator, FSMHallOrderCompleteCh chan elevio.ButtonEvent, CabCopyCh chan [elevio.N_Floors][elevio.N_Buttons]bool) elevator.Elevator { // not finished
	// Implement logic based on e.Config.ClearRequestVariant
	// Update e.Requests accordingly

	switch e.Config.ClearRequestVariant {
	case elevator.CV_All:
		for Button := 0; Button < elevio.N_Buttons; Button++ { // Check this
			e.Requests[e.Floor][Button] = false
		}
		FSMHallOrderCompleteCh <- elevio.ButtonEvent{Floor: e.Floor, Button: elevio.ButtonType(elevio.B_HallUp)}
		FSMHallOrderCompleteCh <- elevio.ButtonEvent{Floor: e.Floor, Button: elevio.ButtonType(elevio.B_HallDown)}

	case elevator.CV_InDoorn:
		e.Requests[e.Floor][elevio.BT_Cab] = false
		switch e.Dirn {
		case elevio.D_Up:
			if !RequestsAbove(e) && !e.Requests[e.Floor][elevio.B_HallUp] {
				e.Requests[e.Floor][elevio.B_HallDown] = false
				FSMHallOrderCompleteCh <- elevio.ButtonEvent{Floor: e.Floor, Button: elevio.ButtonType(elevio.B_HallDown)}
			}
			e.Requests[e.Floor][elevio.B_HallUp] = false
			FSMHallOrderCompleteCh <- elevio.ButtonEvent{Floor: e.Floor, Button: elevio.ButtonType(elevio.B_HallUp)}
		case elevio.D_Down:
			if !RequestsBelow(e) && !e.Requests[e.Floor][elevio.B_HallDown] {
				e.Requests[e.Floor][elevio.B_HallUp] = false
				FSMHallOrderCompleteCh <- elevio.ButtonEvent{Floor: e.Floor, Button: elevio.ButtonType(elevio.B_HallUp)}
			}
			e.Requests[e.Floor][elevio.B_HallDown] = false
			FSMHallOrderCompleteCh <- elevio.ButtonEvent{Floor: e.Floor, Button: elevio.ButtonType(elevio.B_HallDown)}
		case elevio.D_Stop:
			fallthrough
		default:
			e.Requests[e.Floor][elevio.B_HallUp] = false
			e.Requests[e.Floor][elevio.B_HallDown] = false
			FSMHallOrderCompleteCh <- elevio.ButtonEvent{Floor: e.Floor, Button: elevio.ButtonType(elevio.B_HallUp)}
			FSMHallOrderCompleteCh <- elevio.ButtonEvent{Floor: e.Floor, Button: elevio.ButtonType(elevio.B_HallDown)}
		}
	}
	CabCopyCh <- e.Requests
	return e
}
