package hall_request_assigner

import (
	"elevator/elevator"
	"elevator/elevio"
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
	HallRequests [][2]bool               `json:"hallRequests"`
	States       map[string]HRAElevState `json:"states"`
}

type ActiveElevator struct {
	Elevator  elevator.Elevator
	MyAddress string
}

// Consider an array called "ActiveElevators" of objects (or pointers to objects) of type elevio.Elevator, where each elevio.Elevator corresponds to an "active/alive" elevator, that can take requests.
// Make a function HallRequestAssigner that takes in "ActiveElevators" and spits out a similar array of elevio.Elevator objects with newly assigned requests. This array will be fed to the fsm of the individual elevators.

// Init ActiveElevators and append two instances of Elevators to it
// func InitActiveElevators() []ActiveElevator {
//    var ActiveElevators []ActiveElevator
//    ActiveElevators = make([]ActiveElevator, 0)

//    ActiveElevators = append(ActiveElevators, elevio.Elevator{
//        Floor:     1,
//        Dirn:      elevio.DirnUp,                                                                                                                     // Assuming elevio.DirnUp is a valid constant
//        Requests:  [elevio.N_Floors][elevio.N_Buttons]bool{{false, true, false}, {true, false, false}, {false, false, false}, {false, false, false}}, // Example, adjust based on actual N_Floors and N_Buttons
//        Behaviour: elevio.EB_Idle,                                                                                                                    // Replace someBehaviourValue with actual value
//        Config: Config{
//            ClearRequestVariant: CV_All,
//            DoorOpenDurationS:   3.0,
//        },
//        MyAddress: "abc.def.ghi.jkl",
//    })

//    ActiveElevators = append(ActiveElevators, elevio.Elevator{
//        Floor:     1,
//        Dirn:      elevio.DirnUp,                                                                                                                  // Assuming elevio.DirnUp is a valid constant
//        Requests:  [elevio.N_Floors][elevio.N_Buttons]bool{{true, true, false}, {true, true, false}, {true, false, false}, {false, false, false}}, // Example, adjust based on actual N_Floors and N_Buttons
//        Behaviour: elevio.EB_Idle,                                                                                                                 // Replace someBehaviourValue with actual value
//        Config: Config{
//            ClearRequestVariant: CV_All,
//            DoorOpenDurationS:   3.0,
//        },
//        MyAddress: "abc.def.ghi.jkl",
//    })

//    return ActiveElevators
// }

func ActiveElevators_to_CombinedHallRequests(ActiveElevators []ActiveElevator) [][2]bool {
	var CombinedHallRequests [elevio.N_Floors][2]bool

	for i := 0; i < len(ActiveElevators); i++ {
		for floor := 0; floor < elevio.N_Floors; floor++ {
			CombinedHallRequests[floor][0] = CombinedHallRequests[floor][0] || ActiveElevators[i].Elevator.Requests[floor][0]
			CombinedHallRequests[floor][1] = CombinedHallRequests[floor][1] || ActiveElevators[i].Elevator.Requests[floor][1]
		}
	}
	return CombinedHallRequests
}

func ActiveElevators_to_HRAInput(ActiveElevators []ActiveElevator) HRAInput {
	var StateMap map[string]hall_request_assigner.HRAElevState
	for i := 0; i < len(ActiveElevators); i++ {
		StateMap[ActiveElevators[i].MyAddress] = ActiveElevatorsToHRAElevatorState(ActiveElevators[i])
	}

	input := HRAInput{
		HallRequests: ActiveElevators_to_CombinedHallRequests(ActiveElevators),
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

func ActiveElevatorsToHRAElevatorState(ActiveElevator ActiveElevator) HRAElevState {
	var ElevatorState HRAElevState
	ElevatorState.Behavior = BehaviorToString(int(ActiveElevator.Elevator.Behaviour))
	ElevatorState.Floor = ActiveElevator.Elevator.Floor
	ElevatorState.Direction = DirnToString(int(ActiveElevator.Elevator.Dirn))
	ElevatorState.CabRequests = RequestsToCab(ActiveElevator.Elevator.Requests)
	return ElevatorState
}
