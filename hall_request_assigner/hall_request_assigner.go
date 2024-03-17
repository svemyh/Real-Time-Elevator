package hall_request_assigner

import (
	"elevator/elevator"
	"elevator/elevio"
	"encoding/json"
	"fmt"
	"net"
	"os/exec"
	"runtime"
	"strings"
)

// Struct members must be public in order to be accessible by json.Marshal/.Unmarshal
// This means they must start with a capital letter, so we need to use field renaming struct tags to make them camelCase

type HRAElevState struct {
	Behavior    string `json:"behaviour"`
	Floor       int    `json:"floor"`
	Direction   string `json:"direction"`
	CabRequests []bool `json:"cabRequests"`
}

type HRAInput struct {
	HallRequests [elevio.N_Floors][2]bool `json:"hallRequests"`
	States       map[string]HRAElevState  `json:"states"`
}

type ActiveElevator struct {
	Elevator  elevator.Elevator
	MyAddress string
}

// Consider an array called "ActiveElevators" of objects (or pointers to objects) of type elevio.Elevator, where each elevio.Elevator corresponds to an "active/alive" elevator, that can take requests.
// Make a function HallRequestAssigner that takes in "ActiveElevators" and spits out a similar array of elevio.Elevator objects with newly assigned requests. This array will be fed to the fsm of the individual elevators.

func ActiveElevators_to_HRAInput(ActiveElevatorsMap map[string]elevator.Elevator, CombinedHallRequests [elevio.N_Floors][2]bool) HRAInput {

	// ActiveElevatorMap := make(map[string]elevator.Elevator)

	StateMap := make(map[string]HRAElevState)
	for key, elevator := range ActiveElevatorsMap {
		StateMap[key] = ElevatorToHRAElevatorState(elevator)
	}

	input := HRAInput{
		HallRequests: CombinedHallRequests,
		States:       StateMap,
	}
	return input
}

func BehaviorToString(elevatorBehaviour int) string {
	switch elevatorBehaviour {
	case 0:
		return "idle"
	case 1:
		return "doorOpen"
	case 2:
		return "moving"
	default:
		return "error"
	}
}

func DirnToString(dirn int) string {
	switch dirn {
	case -1:
		return "down"
	case 0:
		return "stop"
	case 1:
		return "up"
	default:
		return "error"
	}
}

func RequestsToCab(allRequests [elevio.N_Floors][elevio.N_Buttons]bool) []bool {
	var CabRequests []bool
	for floor := 0; floor < len(allRequests); floor++ {
		CabRequests = append(CabRequests, allRequests[floor][2])
	}
	return CabRequests
}

func ElevatorToHRAElevatorState(Elevator elevator.Elevator) HRAElevState {
	var ElevatorState HRAElevState
	ElevatorState.Behavior = BehaviorToString(int(Elevator.Behaviour))
	ElevatorState.Floor = Elevator.Floor
	ElevatorState.Direction = DirnToString(int(Elevator.Dirn))
	ElevatorState.CabRequests = RequestsToCab(Elevator.Requests)
	return ElevatorState
}

func InitActiveElevator() ActiveElevator {
	ip, _ := LocalIP()
	return ActiveElevator{
		Elevator: elevator.Elevator{
			Floor:     1,
			Dirn:      elevio.D_Stop,
			Behaviour: elevator.EB_Idle,
			Config: elevator.Config{
				ClearRequestVariant: elevator.CV_InDirn,
				DoorOpenDurationS:   3.0,
			},
		},
		MyAddress: ip,
	}
}

func HallRequestAssigner(ActiveElevatorsMap map[string]elevator.Elevator, CombinedHallRequests [elevio.N_Floors][2]bool) map[string][elevio.N_Floors][2]bool {

	hraExecutable := ""
	switch runtime.GOOS {
	case "linux":
		hraExecutable = "hall_request_assigner"
	case "windows":
		hraExecutable = "hall_request_assigner.exe"
	default:
		panic("OS not supported")
	}

	filteredActiveElevatorsMap := ActiveElevatorsMap
/*
	filteredActiveElevatorsMap := make(map[string]elevator.Elevator)
	for ip, elev := range ActiveElevatorsMap {
		if elev.Available { // Check if the elevator is marked as available
			filteredActiveElevatorsMap[ip] = elev
		}
	}
*/
	input := ActiveElevators_to_HRAInput(filteredActiveElevatorsMap, CombinedHallRequests)

	jsonBytes, err := json.Marshal(input)
	if err != nil {
		fmt.Println("json.Marshal error: ", err)
		//return "Error parsing input to json"
	}

	ret, err := exec.Command("hall_request_assigner/"+hraExecutable, "-i", string(jsonBytes)).CombinedOutput()
	if err != nil {
		fmt.Println("exec.Command error: ", err)
		fmt.Println(string(ret))
		//return "Error executing hall_request_assigner"
	}

	output := new(map[string][elevio.N_Floors][2]bool)
	err = json.Unmarshal(ret, &output)
	if err != nil {
		fmt.Println("json.Unmarshal error: ", err)
		//return "Error parsing output from json"
	}

	return *output
}

func LocalIP() (string, error) {
	var localIP string

	if localIP == "" {
		conn, err := net.DialTCP("tcp4", nil, &net.TCPAddr{IP: []byte{8, 8, 8, 8}, Port: 53})
		if err != nil {
			return "", err
		}
		defer conn.Close()
		localIP = strings.Split(conn.LocalAddr().String(), ":")[0]
	}
	return localIP, nil
}
