package main

import (
	"context"
	"elevator/elevio"
	"elevator/fsm"
	"elevator/network"
	"fmt"
)

func main() {
	elevio.Init("localhost:15657", elevio.N_Floors)

	fmt.Printf("Started!\n")

	device := elevio.ElevInputDevice{
		FloorSensorCh:   make(chan int),
		RequestButtonCh: make(chan elevio.ButtonEvent),
		StopButtonCh:    make(chan bool),
		ObstructionCh:   make(chan bool),
	}
	//fsm_terminate := make(chan, bool)

	go elevio.PollFloorSensor(device.FloorSensorCh)
	go elevio.PollButtons(device.RequestButtonCh)
	go elevio.PollStopButton(device.StopButtonCh)
	go elevio.PollObstructionSwitch(device.ObstructionCh)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go network.InitNetwork(ctx) // Alias: RunPrimaryBackup()

	go fsm.FsmRun(device)

	go network.RestartOnReconnect()

	//establishConnectionWithPrimary() // TCP

	select {}

}
