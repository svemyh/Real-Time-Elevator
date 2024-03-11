package main

import (
	"elevator/elevio"
	"elevator/fsm"
	//"elevator/hall_request_assigner"
	"elevator/network"
	"fmt"
)



func main() {
	elevio.Init("localhost:15657", elevio.N_Floors)

	fmt.Printf("Started!\n")

	Channels := network.NewElevatorSystemChannels()

	device := elevio.ElevInputDevice{
		FloorSensorCh:   make(chan int),
		RequestButtonCh: make(chan elevio.ButtonEvent),
		StopButtonCh:    make(chan bool),
		ObstructionCh:   make(chan bool),
	}

	/* CLEANUP. Moved to network
	FSMStateUpdateCh := make(chan hall_request_assigner.ActiveElevator, 1024)
	FSMHallOrderCompleteCh := make(chan elevio.ButtonEvent, 1024)
	StateUpdateCh := make(chan hall_request_assigner.ActiveElevator, 1024)
	HallOrderCompleteCh := make(chan elevio.ButtonEvent, 1024)
	DisconnectedElevatorCh := make(chan string, 1024)
	FSMAssignedHallRequestsCh := make(chan [elevio.N_Floors][elevio.N_Buttons - 1]bool, 1024)
	AssignHallRequestsCh := make(chan map[string][elevio.N_Floors][elevio.N_Buttons - 1]bool, 1024)
	*/
	//fsm_terminate := make(chan, bool)

	go elevio.PollFloorSensor(device.FloorSensorCh)
	go elevio.PollButtons(device.RequestButtonCh)
	go elevio.PollStopButton(device.StopButtonCh)
	go elevio.PollObstructionSwitch(device.ObstructionCh)

	go network.InitNetwork(Channels.FSMStateUpdateCh, Channels.FSMHallOrderCompleteCh, Channels.StateUpdateCh, Channels.HallOrderCompleteCh, Channels.DisconnectedElevatorCh, Channels.FSMAssignedHallRequestsCh, Channels.AssignHallRequestsMapCh) // Alias: RunPrimaryBackup()

	go fsm.FsmRun(device, Channels.FSMStateUpdateCh, Channels.FSMHallOrderCompleteCh, Channels.FSMAssignedHallRequestsCh) // should also pass in the folowing as arguments at some point: (FSMStateUpdateCh chan hall_request_assigner.ActiveElevator, FSMHallOrderCompleteCh chan elevio.ButtonEvent)

	go network.RestartOnReconnect()

	//establishConnectionWithPrimary() // TCP

	select {}

}
