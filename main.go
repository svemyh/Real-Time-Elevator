package main

import (
	"elevator/elevator"
	"time"
)

const inputPollRate = 25 * time.Millisecond

func main() {
	// elevio.Init("localhost:15657", elevio.N_Floors)

	// // Initialize the elevator to a known state
	// if elevio.GetFloor() == -1 {
	// 	fsm.FsmOnInitBetweenFloors()
	// }

	// // Main loop
	// for {
	// 	// Poll request buttons
	// 	for f := 0; f < elevio.N_Floors; f++ {
	// 		for b := elevio.BT_HallUp; b <= elevio.BT_Cab; b++ {
	// 			if v := elevio.GetButton(b, f); v {
	// 				fsm.FsmOnRequestButtonPress(f, b)
	// 			}
	// 		}
	// 	}

	// 	// Poll floor sensor
	// 	if f := elevio.GetFloor(); f != -1 {
	// 		fsm.FsmOnFloorArrival(f)
	// 	}

	// 	// Handle door timeout
	// 	if timer.TimerTimedOut() {
	// 		timer.TimerStop()
	// 		fsm.FsmOnDoorTimeout()
	// 	}

	// 	time.Sleep(inputPollRate)
	// }

	var elev elevator.Elevator
	elev.Floor = 1
	elev.Dirn = D_Up
	elev.Requests[0][1] = 1                  // Example request
	elev.Behaviour = EB_Idle                 /* ElevatorBehaviour value */
	elev.Config.ClearRequestVariant = CV_All /* ClearRequestVariant value */
	elev.Config.DoorOpenDurationS = 3.0

	device := NewElevInputDevice()
}
