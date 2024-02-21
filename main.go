package main

import (
	"elevator/elevio"
	"elevator/fsm"
	"fmt"
	"time"
)

const inputPollRate = 25 * time.Millisecond

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

	
	fsm.FsmRun(device) 

}
