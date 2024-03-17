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

func ChooseDirection(e elevator.Elevator) (elevio.Dirn, elevator.ElevatorBehaviour) {
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
		return e.Requests[e.Floor][elevio.BT_HallDown] || e.Requests[e.Floor][elevio.BT_Cab] || !requestsBelow(e)
	case elevio.D_Up:
		return e.Requests[e.Floor][elevio.BT_HallUp] || e.Requests[e.Floor][elevio.BT_Cab] || !requestsAbove(e)
	default:
		return true
	}
}

func ShouldClearImmediately(e elevator.Elevator, btnFloor int, btnType elevio.ButtonType) bool { 
	switch e.Config.ClearRequestVariant {
	case elevator.CV_All:
		return e.Floor == btnFloor
	case elevator.CV_InDirn:
		return e.Floor == btnFloor && ((e.Dirn == elevio.D_Up && btnType == elevio.BT_HallUp) ||
			(e.Dirn == elevio.D_Down && btnType == elevio.BT_HallDown) ||
			e.Dirn == elevio.D_Stop || btnType == elevio.BT_Cab)
	default:
		return false
	}
}

func ClearAtCurrentFloorAndSendCompletedOrder(e 						elevator.Elevator, 
						 					  FSMHallOrderCompleteCh chan<- elevio.ButtonEvent, 
						 					  CabCopyCh 				chan<- [elevio.N_Floors][elevio.N_Buttons]bool,
) elevator.Elevator { 
	switch e.Config.ClearRequestVariant {
	case elevator.CV_All:
		for button := 0; button < elevio.N_Buttons; button++ { 
			e.Requests[e.Floor][button] = false
		}
		FSMHallOrderCompleteCh <- elevio.ButtonEvent{Floor: e.Floor, Button: elevio.ButtonType(elevio.BT_HallUp)}
		FSMHallOrderCompleteCh <- elevio.ButtonEvent{Floor: e.Floor, Button: elevio.ButtonType(elevio.BT_HallDown)}

	case elevator.CV_InDirn:
		e.Requests[e.Floor][elevio.BT_Cab] = false
		switch e.Dirn {
		case elevio.D_Up:
			if !requestsAbove(e) && !e.Requests[e.Floor][elevio.BT_HallUp] {
				e.Requests[e.Floor][elevio.BT_HallDown] = false
				FSMHallOrderCompleteCh <- elevio.ButtonEvent{Floor: e.Floor, Button: elevio.ButtonType(elevio.BT_HallDown)}
			}
			e.Requests[e.Floor][elevio.BT_HallUp] = false
			FSMHallOrderCompleteCh <- elevio.ButtonEvent{Floor: e.Floor, Button: elevio.ButtonType(elevio.BT_HallUp)}
		case elevio.D_Down:
			if !requestsBelow(e) && !e.Requests[e.Floor][elevio.BT_HallDown] {
				e.Requests[e.Floor][elevio.BT_HallUp] = false
				FSMHallOrderCompleteCh <- elevio.ButtonEvent{Floor: e.Floor, Button: elevio.ButtonType(elevio.BT_HallUp)}
			}
			e.Requests[e.Floor][elevio.BT_HallDown] = false
			FSMHallOrderCompleteCh <- elevio.ButtonEvent{Floor: e.Floor, Button: elevio.ButtonType(elevio.BT_HallDown)}
		case elevio.D_Stop:
			fallthrough
		default:
			e.Requests[e.Floor][elevio.BT_HallUp] = false
			e.Requests[e.Floor][elevio.BT_HallDown] = false
			FSMHallOrderCompleteCh <- elevio.ButtonEvent{Floor: e.Floor, Button: elevio.ButtonType(elevio.BT_HallUp)}
			FSMHallOrderCompleteCh <- elevio.ButtonEvent{Floor: e.Floor, Button: elevio.ButtonType(elevio.BT_HallDown)}
		}
	}
	CabCopyCh <- e.Requests
	return e
}