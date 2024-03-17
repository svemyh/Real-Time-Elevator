package main

import (
	"elevator/elevio"
	"elevator/fsm"
	"elevator/primary_backup"
	"fmt"
	"os"
	"time"
)

func main() {
	fmt.Printf("Started!\n")

	time.Sleep(5 * time.Second)
	var InitCabCopy [elevio.N_Floors]bool
	if len(os.Args) == 2 {
		InitCabCopy = elevio.StringToCabArray(os.Args[1])
	}
	elevio.Init("localhost:15657", elevio.N_Floors)


	Channels := primary_backup.NewElevatorSystemChannels()
	CabCopyCh := make(chan [elevio.N_Floors][elevio.N_Buttons]bool)

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

	go primary_backup.InitPrimaryBackup(Channels.FSMStateUpdateCh, Channels.FSMHallOrderCompleteCh, Channels.StateUpdateCh, Channels.HallOrderCompleteCh, Channels.DisconnectedElevatorCh, Channels.FSMAssignedHallRequestsCh, Channels.AssignHallRequestsMapCh, Channels.AckCh) // Alias: RunPrimaryBackup()

	go fsm.LocalElevatorFSM(device, Channels.FSMStateUpdateCh, Channels.FSMHallOrderCompleteCh, Channels.FSMAssignedHallRequestsCh, CabCopyCh, InitCabCopy)

	go primary_backup.RestartOnReconnect(CabCopyCh)

	go primary_backup.UDPReadCombinedHallRequests(primary_backup.HALL_LIGHTS_PORT)
	go primary_backup.UDPBroadcastAlive(primary_backup.LOCAL_ELEVATOR_ALIVE_PORT)

	select {}

}
