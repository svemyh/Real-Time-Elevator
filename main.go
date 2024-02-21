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

	if f := elevio.GetFloor(); f == -1 {
		fsm.FsmOnInitBetweenFloors()
	}

	// Polling for new actions/events of the system.
	for {
		select {
		case floor := <-device.FloorSensorCh:
			fmt.Println("Floor Sensor:", floor)
			//var prev static int = 0
			if floor != -1 && floor != prev {
				fmt.Println("Floor Sensor1:")
				fsm.FsmOnFloorArrival(floor)
			}
			fmt.Println("Floor Sensor2:")
			prev = floor
			break
		case buttonEvent := <-device.RequestButtonCh:
			fmt.Println("Button Pressed:", buttonEvent)
			fsm.FsmOnRequestButtonPress(buttonEvent.Floor, elevio.Button(buttonEvent.Button))
			break
		case obstructionSignal := <-device.ObstructionCh:
			fmt.Println("Obstruction Detected", obstructionSignal)
			break

		default:
			// No action - prevents blocking on channel reads
			time.Sleep(500 * time.Millisecond)
		}
		if timer.TimerTimedOut() { // should be reworked into a channel
			timer.TimerStop()
			fsm.FsmOnDoorTimeout()
		}
	}
}
