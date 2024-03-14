package main

import (
	"elevator/elevio"
	"elevator/fsm"
	"os"
	"elevator/network"
	"fmt"
)

func main() {
	//just to enable running multiple elev server from same computer by doing go run main() port 
	port := "15657"
	if len(os.Args) == 2 {
		port = os.Args[1]
	}
	elevio.Init("localhost:"+port, elevio.N_Floors)

	fmt.Printf("Started!\n")

	Channels := network.NewElevatorSystemChannels()

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


	go fsm.FsmRun(device, Channels.FSMStateUpdateCh, Channels.FSMHallOrderCompleteCh, Channels.FSMAssignedHallRequestsCh) // should also pass in the folowing as arguments at some point: (FSMStateUpdateCh chan hall_request_assigner.ActiveElevator, FSMHallOrderCompleteCh chan elevio.ButtonEvent)

	go network.RestartOnReconnect()

	go network.UDPReadCombinedHallRequests(network.HALL_LIGHTS_PORT)

	select {}

}
