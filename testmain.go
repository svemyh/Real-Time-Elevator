package main

import (
	"elevator/elevio"
	"elevator/hall_request_assigner"
	"elevator/network"
	"fmt"
	"time"
)

func main() {
	// -------- DEMO-code: hall_request_assigner -------- //

	my_activeElev := hall_request_assigner.InitActiveElevator() // Initializes a default elevator with floor 1, direction stop, and behavior idle. This is the elevator that will be used to test the hall_request_assigner functionality.

	//var ActiveElevators []hall_request_assigner.ActiveElevator // Initializes ActiveElevators.
	//ActiveElevators = append(ActiveElevators, activeElev)      // Adds an elevator to ActiveElevators.

	// The following []ActiveElevator object contains updated hall requests determined by hall_request_assigner.exe. Not certain if cab-requests should or should not be updated aswell. Question: Should we clear the cab requests here? - Answer depends on how hall_request_assigner.exe is coded. TODO: Check functionality by enabling/disabling this at later time when functioning elevators are acheived.
	//NewActiveElevators := hall_request_assigner.HallRequestAssigner(ActiveElevators) // Computes new hall requests for the elevators in ActiveElevators. Returns object of same type as ActiveElevators.
	//fmt.Println("NewActiveElevators:", NewActiveElevators)

	// -------- Testing READ/WRITE over TCP -------- //

	//greetingMsg := Greeting{Type: TypeGreeting, ID: 1, Message: "Hello, TCP!"}
	my_ActiveElevatorMsg := network.MsgActiveElevator{
		Type:    network.TypeActiveElevator,
		Content: my_activeElev,
	}

	my_ButtonEvent := elevio.ButtonEvent{
		Floor:  2,
		Button: elevio.BT_HallUp,
	}

	my_ButtonEventMsg := network.MsgButtonEvent{
		Type:    network.TypeButtonEvent,
		Content: my_ButtonEvent,
	}

	fmt.Println("Sending an ActiveElevator to primary now..")
	network.StartClient(network.TCP_LISTEN_PORT, my_ActiveElevatorMsg)
	fmt.Println("ActiveElevator has jsut been sendt to primary. Sleeping 3 sec..")
	time.Sleep(3 * time.Second)

	fmt.Println("Sending an ButtonEvent to primary now..")
	network.StartClient(network.TCP_LISTEN_PORT, my_ButtonEventMsg)
	fmt.Println("ButtonEvent has jsut been sendt to primary. Sleeping 3 sec..")
	time.Sleep(3 * time.Second)

}