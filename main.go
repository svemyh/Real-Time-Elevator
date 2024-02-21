package main

import (
	"elevator/elevio"
	"elevator/fsm"
	"elevator/hall_request_assigner"
	"encoding/json"
	"fmt"
	"os/exec"
	"runtime"
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
	//fsm_terminate := make(chan, bool)

	go elevio.PollFloorSensor(device.FloorSensorCh)
	go elevio.PollButtons(device.RequestButtonCh)
	go elevio.PollStopButton(device.StopButtonCh)
	go elevio.PollObstructionSwitch(device.ObstructionCh)

	go fsm.FsmRun(device)

	hraExecutable := ""
	switch runtime.GOOS {
	case "linux":
		hraExecutable = "hall_request_assigner"
	case "windows":
		hraExecutable = "hall_request_assigner.exe"
	default:
		panic("OS not supported")
	}

	input := hall_request_assigner.HRAInput{
		HallRequests: [][2]bool{{false, false}, {true, false}, {false, false}, {false, true}},
		States: map[string]hall_request_assigner.HRAElevState{
			"one": hall_request_assigner.HRAElevState{
				Behavior:    "moving",
				Floor:       2,
				Direction:   "up",
				CabRequests: []bool{false, false, false, true},
			},
			"two": hall_request_assigner.HRAElevState{
				Behavior:    "idle",
				Floor:       0,
				Direction:   "stop",
				CabRequests: []bool{false, false, false, false},
			},
			"three": hall_request_assigner.HRAElevState{
				Behavior:    "idle",
				Floor:       0,
				Direction:   "stop",
				CabRequests: []bool{false, false, false, false},
			},
		},
	}

	jsonBytes, err := json.Marshal(input)
	if err != nil {
		fmt.Println("json.Marshal error: ", err)
		return
	}

	ret, err := exec.Command("hall_request_assigner/"+hraExecutable, "-i", string(jsonBytes)).CombinedOutput()
	if err != nil {
		fmt.Println("exec.Command error: ", err)
		fmt.Println(string(ret))
		return
	}

	output := new(map[string][][2]bool)
	err = json.Unmarshal(ret, &output)
	if err != nil {
		fmt.Println("json.Unmarshal error: ", err)
		return
	}

	fmt.Printf("output: \n")
	for k, v := range *output {
		fmt.Printf("%6v :  %+v\n", k, v)
	}

	select {}

	// myState := InitStateByBroadcastingNetworkAndWait()
	// If myState == PRIMARY:
	//		ActiveElevators <- getAllElevatorStates()
	// 		sendOverNetworkToSecondary(ActiveElevators)
	//		for {
	//			ack <- network_recieved_ack
	//			time.sleep()
	//		}
	//      NewElevatorOrders = HallRequestAssigner(ActiveElevators)
	//		DistributeOrdersOverNetwork(NewElevatorOrders)
	//
	// func DistributeOrdersOverNetwork(NewElevatorOrders):
	//
	// 		for i in ActiveElevators:
	//			sendOverNetwork(ActiveElevators[i])
	//			for {
	//				ack <- network_reciever
	//				time.sleep()
	//			}
	//
	//		for i in ActiveElevators:
	//			sendOverNetwork(buttonlights)
	//			for {
	//				ack <- network_reciever
	//				time.sleep()
	//			}
	//
	//
	// chan networkReciever
	// func getAllElevatorStates(chan networkReciever)

	// case newEvent := <- networkReciever
	//	 (MYIP, elevio.elevator)
	//   ActiveElevators("MYIP") = elevio.elevator

}
