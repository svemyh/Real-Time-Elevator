package main

import (
	"elevator/elevio"
	"elevator/fsm"
	"elevator/timer"
	"fmt"
	"time"
)

const inputPollRate = 25 * time.Millisecond

var prev int = -1

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
	// 		for b := elevio.BT_HallUp; b <= elevio.BT_Cab; b++ {	// 	// Handle door timeout
	// 	if timer.TimerTimedOut() {
	// 		timer.TimerStop()
	// 		fsm.FsmOnDoorTimeout()
	// 	}
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

	// var elev elevator.Elevator
	// elev.Floor = 1
	// elev.Dirn = elevio.D_Up
	// elev.Requests[0][1] = 1                           // Example request
	// elev.Behaviour = elevator.EB_Idle                 /* ElevatorBehaviour value */
	// elev.Config.ClearRequestVariant = elevator.CV_All /* ClearRequestVariant value */
	// elev.Config.DoorOpenDurationS = 3.0
	//var device elevio.ElevInputDevice

	elevio.Init("localhost:15657", elevio.N_Floors)

	fmt.Printf("Started!\n")

	device := elevio.ElevInputDevice{
		FloorSensorCh:   make(chan int),
		RequestButtonCh: make(chan elevio.ButtonEvent),
		StopButtonCh:    make(chan bool),
		ObstructionCh:   make(chan bool),
	}

	go elevio.PollFloorSensor(device.FloorSensorCh)
	go elevio.PollButtons(device.RequestButtonCh)
	go elevio.PollStopButton(device.StopButtonCh)
	go elevio.PollObstructionSwitch(device.ObstructionCh)

	// TODO(?): if input.floorSensor() == -1 {fsm_onInitBetweenFloors()}

	// Run for a short period to demonstrate receiving signals. Check!
	// endTime := time.Now().Add(10 * time.Second)
	// for time.Now().Before(endTime) {

	if f := elevio.GetFloor(); f == -1 {
		fsm.FsmOnInitBetweenFloors()
	}

	for {
		// Poll request buttons
		for f := 0; f < elevio.N_Floors; f++ {
			// for b := elevio.BT_HallUp; b <= elevio.BT_Cab; b++ {
			for b := elevio.BT_HallUp; b <= elevio.BT_Cab; b++ {
				if v := elevio.GetButton(b, f); v {
					fsm.FsmOnRequestButtonPress(f, elevio.Button(b))
				}
			}
		}

		// Poll floor sensor

		flr := elevio.GetFloor() 
        if flr != -1 && flr != prev {
            fsm.FsmOnFloorArrival(flr) 
            if timer.TimerTimedOut() {
                timer.TimerStop()
                fsm.FsmOnDoorTimeout()
            }
        }
        prev = flr

		// Handle door timeout
		if timer.TimerTimedOut() {
			timer.TimerStop()
			fsm.FsmOnDoorTimeout()
		}

		// for {
		// 	select {
		// 	case floor := <-device.FloorSensorCh:
		// 		fmt.Println("Floor Sensor:", floor)
		// 		break
		// 	case buttonEvent := <-device.RequestButtonCh:
		// 		fmt.Println("Button Pressed:", buttonEvent)
		// 		break
		// 	case stopSignal := <-device.StopButtonCh:
		// 		fmt.Println("Stop Button Pressed", stopSignal)
		// 		break
		// 	case obstructionSignal := <-device.ObstructionCh:
		// 		fmt.Println("Obstruction Detected", obstructionSignal)
		// 		break
		// 	default:
		// 		// No action - prevents blocking on channel reads
		// 		time.Sleep(500 * time.Millisecond)
		// 	}
		// }
	}
}