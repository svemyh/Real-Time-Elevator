package main

import (
	"elevator/elevio"
	"elevator/fsm"
	"elevator/network"
	"fmt"
)

func main() {
	elevio.Init("localhost:15657", elevio.N_Floors)

	fmt.Printf("Started!\n")

	ElevatorChannels := network.NewElevatorSystemChannels()
	FSMChannels := network.NewFSMSystemChannels()

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

	go network.InitNetwork(FSMChannels, ElevatorChannels) // Alias: RunPrimaryBackup()

	go fsm.FsmRun(device, FSMChannels) // should also pass in the folowing as arguments at some point: (FSMStateUpdateCh chan hall_request_assigner.ActiveElevator, FSMHallOrderCompleteCh chan elevio.ButtonEvent)

	go network.RestartOnReconnect()

	go network.UDPReadCombinedHallRequests(network.HALL_LIGHTS_PORT)

	//establishConnectionWithPrimary() // TCP

	select {}

}
