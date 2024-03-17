package requests

import (
	"elevator/elevator"
	"elevator/elevio"
)

func requestsAbove(e elevator.Elevator) bool {
	for f := e.Floor + 1; f < elevio.N_Floors; f++ {
		for btn := range e.Requests[f] {
			if e.Requests[f][btn] {
				return true
			}
		}
	}
	return false
}

func requestsBelow(e elevator.Elevator) bool {
	for f := 0; f < e.Floor; f++ {
		for btn := range e.Requests[f] {
			if e.Requests[f][btn] {
				return true
			}
		}
	}
	return false
}

func requestsHere(e elevator.Elevator) bool {
	for btn := range e.Requests[e.Floor] {
		if e.Requests[e.Floor][btn] {
			return true
		}
	}
	return false
}

func HasRequests(e elevator.Elevator) bool {
	if requestsAbove(e) || requestsBelow(e) || requestsHere(e) {
		return false
	}
	return true
}

func ChooseDirection(e elevator.Elevator) (elevio.Dirn, elevator.ElevatorBehaviour) { // Elevator doesn't have a motorDirection, but it does have a Dirn
	switch {
	case requestsAbove(e):
		e.Dirn = elevio.D_Up
		return elevio.D_Up, elevator.EB_Moving
	case requestsBelow(e):
		e.Dirn = elevio.D_Down
		return elevio.D_Down, elevator.EB_Moving
	case requestsHere(e):
		e.Dirn = elevio.D_Stop
		return elevio.D_Stop, elevator.EB_DoorOpen
	default:
		return elevio.D_Stop, elevator.EB_Idle
	}
}

func ShouldStop(e elevator.Elevator) bool {
	switch e.Dirn {
	case elevio.D_Down:
		return e.Requests[e.Floor][elevio.B_HallDown] || e.Requests[e.Floor][elevio.B_Cab] || !requestsBelow(e)
	case elevio.D_Up:
		return e.Requests[e.Floor][elevio.B_HallUp] || e.Requests[e.Floor][elevio.B_Cab] || !requestsAbove(e)
	default:
		return true
	}
}

func ShouldClearImmediately(e elevator.Elevator, btnFloor int, btnType elevio.Button) bool { 
	// Assuming you have a configuration for ClearRequestVariant
	switch e.Config.ClearRequestVariant {
	case elevator.CV_All:
		return e.Floor == btnFloor
	case elevator.CV_InDirn:
		return e.Floor == btnFloor && ((e.Dirn == elevio.D_Up && btnType == elevio.B_HallUp) ||
			(e.Dirn == elevio.D_Down && btnType == elevio.B_HallDown) ||
			e.Dirn == elevio.D_Stop || btnType == elevio.B_Cab)
	default:
		return false
	}
}

func ClearAtCurrentFloorAndSendUpdate(e 						elevator.Elevator, 
						 FSMHallOrderCompleteCh chan<- elevio.ButtonEvent, 
						 CabCopyCh 				chan<- [elevio.N_Floors][elevio.N_Buttons]bool,
) elevator.Elevator { 
	switch e.Config.ClearRequestVariant {
	case elevator.CV_All:
		for button := 0; button < elevio.N_Buttons; button++ { 
			e.Requests[e.Floor][button] = false
		}
		FSMHallOrderCompleteCh <- elevio.ButtonEvent{Floor: e.Floor, Button: elevio.ButtonType(elevio.B_HallUp)}
		FSMHallOrderCompleteCh <- elevio.ButtonEvent{Floor: e.Floor, Button: elevio.ButtonType(elevio.B_HallDown)}

	case elevator.CV_InDirn:
		e.Requests[e.Floor][elevio.BT_Cab] = false
		switch e.Dirn {
		case elevio.D_Up:
			if !requestsAbove(e) && !e.Requests[e.Floor][elevio.B_HallUp] {
				e.Requests[e.Floor][elevio.B_HallDown] = false
				FSMHallOrderCompleteCh <- elevio.ButtonEvent{Floor: e.Floor, Button: elevio.ButtonType(elevio.B_HallDown)}
			}
			e.Requests[e.Floor][elevio.B_HallUp] = false
			FSMHallOrderCompleteCh <- elevio.ButtonEvent{Floor: e.Floor, Button: elevio.ButtonType(elevio.B_HallUp)}
		case elevio.D_Down:
			if !requestsBelow(e) && !e.Requests[e.Floor][elevio.B_HallDown] {
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

/*

func ClearAtCurrentFloor(e elevator.Elevator, 
	FSMHallOrderCompleteCh chan<- elevio.ButtonEvent, 
	CabCopyCh chan<- [elevio.N_Floors][elevio.N_Buttons]bool) elevator.Elevator {
// Clear all requests at the current floor based on the elevator's configuration.
switch e.Config.ClearRequestVariant {
	case elevator.CV_All:
	for button := 0; button < elevio.N_Buttons; button++ {
		e.Requests[e.Floor][button] = false
	}
	notifyClearedRequests(e.Floor, FSMHallOrderCompleteCh)

	case elevator.CV_InDirn:
		e.Requests[e.Floor][elevio.BT_Cab] = false

		clearDirectionalRequests(e, FSMHallOrderCompleteCh)
	}
		CabCopyCh <- e.Requests
		return e
	}

func notifyClearedRequests(floor int, ch chan<- elevio.ButtonEvent) {
	ch <- elevio.ButtonEvent{Floor: floor, Button: elevio.ButtonType(elevio.B_HallUp)}
	ch <- elevio.ButtonEvent{Floor: floor, Button: elevio.ButtonType(elevio.B_HallDown)}
}

func clearDirectionalRequests(e elevator.Elevator, ch chan<- elevio.ButtonEvent) {
switch e.Dirn {
	case elevio.D_Up:
		clearIfNoRequestsAbove(e, ch, elevio.B_HallDown)
		e.Requests[e.Floor][elevio.B_HallUp] = false
		ch <- elevio.ButtonEvent{Floor: e.Floor, Button: elevio.ButtonType(elevio.B_HallUp)}

	case elevio.D_Down:
		clearIfNoRequestsBelow(e, ch, elevio.B_HallUp)
		e.Requests[e.Floor][elevio.B_HallDown] = false
		ch <- elevio.ButtonEvent{Floor: e.Floor, Button: elevio.ButtonType(elevio.B_HallDown)}

	case elevio.D_Stop:
		e.Requests[e.Floor][elevio.B_HallUp] = false
		e.Requests[e.Floor][elevio.B_HallDown] = false
		notifyClearedRequests(e.Floor, ch)
		}
}

func clearIfNoRequestsAbove(e elevator.Elevator, ch chan<- elevio.ButtonEvent, btnType elevio.Button) {
	if !requestsAbove(e) && !e.Requests[e.Floor][btnType] {
	e.Requests[e.Floor][btnType] = false
	ch <- elevio.ButtonEvent{Floor: e.Floor, Button: elevio.ButtonType(btnType)}
	}
}

func clearIfNoRequestsBelow(e elevator.Elevator, ch chan<- elevio.ButtonEvent, btnType elevio.Button) {
	if !requestsBelow(e) && !e.Requests[e.Floor][btnType] {
	e.Requests[e.Floor][btnType] = false
	ch <- elevio.ButtonEvent{Floor: e.Floor, Button: elevio.ButtonType(btnType)}
	}
}
*/
