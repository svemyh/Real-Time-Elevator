package main

import (
	"elevator/elevio"
	"elevator/fsm"
	"elevator/network"
	"fmt"
	"os"
)

func main() {
	//just to enable running multiple elev server from same computer by doing go run main() port
	//"15657" default port for elev server
	var InitCabCopy [elevio.N_Floors]bool
	if len(os.Args) == 2 {
		InitCabCopy = elevio.StringToCabArray(os.Args[1])
	}
	elevio.Init("localhost:15657", elevio.N_Floors)

	fmt.Printf("Started!\n")
	fmt.Println("The cab copy is: ", InitCabCopy)

	Channels := network.NewElevatorSystemChannels()
	CabCopyCh := make(chan [elevio.N_Floors][elevio.N_Buttons]bool) //no buffer in order to make sending blockig

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

	go network.InitNetwork(Channels.FSMStateUpdateCh, Channels.FSMHallOrderCompleteCh, Channels.StateUpdateCh, Channels.HallOrderCompleteCh, Channels.DisconnectedElevatorCh, Channels.FSMAssignedHallRequestsCh, Channels.AssignHallRequestsMapCh, Channels.AckCh) // Alias: RunPrimaryBackup()
	// REFACTOR: Can be moved to InitNetwork()?

	//run local elevator
	go fsm.FsmRun(device, Channels.FSMStateUpdateCh, Channels.FSMHallOrderCompleteCh, Channels.FSMAssignedHallRequestsCh, CabCopyCh, InitCabCopy) // should also pass in the folowing as arguments at some point: (FSMStateUpdateCh chan hall_request_assigner.ActiveElevator, FSMHallOrderCompleteCh chan elevio.ButtonEvent)

	go network.RestartOnReconnect(CabCopyCh)

	go network.UDPReadCombinedHallRequests(network.HALL_LIGHTS_PORT)

	//establishConnectionWithPrimary() // TCP

	select {}

}
