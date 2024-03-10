package main

import (
	"context"
	"elevator/elevio"
	"elevator/fsm"
	"elevator/hall_request_assigner"
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

	FSMStateUpdateCh := make(chan hall_request_assigner.ActiveElevator, 1024)
	FSMHallOrderCompleteCh := make(chan elevio.ButtonEvent, 1024)
	StateUpdateCh := make(chan hall_request_assigner.ActiveElevator, 1024)
	HallOrderCompleteCh := make(chan elevio.ButtonEvent, 1024)
	DisconnectedElevatorCh := make(chan string, 1024)
	FSMAssignedHallRequestsCh := make(chan [elevio.N_Floors][elevio.N_Buttons - 1]bool, 1024)
	AssignHallRequestsCh := make(chan map[string][elevio.N_Floors][elevio.N_Buttons - 1]bool, 1024)

	//fsm_terminate := make(chan, bool)

	go elevio.PollFloorSensor(device.FloorSensorCh)
	go elevio.PollButtons(device.RequestButtonCh)
	go elevio.PollStopButton(device.StopButtonCh)
	go elevio.PollObstructionSwitch(device.ObstructionCh)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go network.InitNetwork(ctx, FSMStateUpdateCh, FSMHallOrderCompleteCh, StateUpdateCh, HallOrderCompleteCh, DisconnectedElevatorCh, FSMAssignedHallRequestsCh, AssignHallRequestsCh) // Alias: RunPrimaryBackup()

	go fsm.FsmRun(device, FSMStateUpdateCh, FSMHallOrderCompleteCh, FSMAssignedHallRequestsCh) // should also pass in the folowing as arguments at some point: (FSMStateUpdateCh chan hall_request_assigner.ActiveElevator, FSMHallOrderCompleteCh chan elevio.ButtonEvent)

	go network.RestartOnReconnect()

	//establishConnectionWithPrimary() // TCP

	select {}

}
