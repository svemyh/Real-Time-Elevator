//DRAFT FILE FOR MAIN

package main

import (
	"elevator/elevio"
	"elevator/fsm"
	"elevator/network"
	
	"fmt"
)

func main() {
	//INIT AND RUN LOCAL FSM
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
	go fsm.FsmRun(device)

	
	//Init all necessary channels to run RunPrimaryBackup routine here
	go RunPrimaryBackup(necessarychannels...)



