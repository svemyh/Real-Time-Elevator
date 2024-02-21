package hall_request_assigner

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

// Consider an array called "ActiveElevators" of objects (or pointers to objects) of type elevio.Elevator, where each elevio.Elevator corresponds to an "active/alive" elevator, that can take requests.
// Make a function HallRequestAssigner that takes in "ActiveElevators" and spits out a similar array of elevio.Elevator objects with newly assigned requests. This array will be fed to the fsm of the individual elevators.




// // Init ActiveElevators and append two instances of Elevators to it
// func InitActiveElevators() []elevio.Elevator {
// 	var ActiveElevators []elevio.Elevator
// 	ActiveElevators = make([]elevio.Elevator, 0)

// 	ActiveElevators = append(ActiveElevators, elevio.Elevator{
// 		Floor:     1,
// 		Dirn:      elevio.DirnUp,                                                                                                                     // Assuming elevio.DirnUp is a valid constant
// 		Requests:  [elevio.N_Floors][elevio.N_Buttons]bool{{false, true, false}, {true, false, false}, {false, false, false}, {false, false, false}}, // Example, adjust based on actual N_Floors and N_Buttons
// 		Behaviour: elevio.EB_Idle,                                                                                                                    // Replace someBehaviourValue with actual value
// 		Config: Config{
// 			ClearRequestVariant: CV_All,
// 			DoorOpenDurationS:   3.0,
// 		},
// 	})

// 	ActiveElevators = append(ActiveElevators, elevio.Elevator{
// 		Floor:     1,
// 		Dirn:      elevio.DirnUp,                                                                                                                  // Assuming elevio.DirnUp is a valid constant
// 		Requests:  [elevio.N_Floors][elevio.N_Buttons]bool{{true, true, false}, {true, true, false}, {true, false, false}, {false, false, false}}, // Example, adjust based on actual N_Floors and N_Buttons
// 		Behaviour: elevio.EB_Idle,                                                                                                                 // Replace someBehaviourValue with actual value
// 		Config: Config{
// 			ClearRequestVariant: CV_All,
// 			DoorOpenDurationS:   3.0,
// 		},
// 	})

// 	ActiveElevators = append(ActiveElevators, ElevatorInit())
// 	return ActiveElevators
// }

// func ActiveElevators_to_HRAInput(ActiveElevators []elevio.Elevator) [elevio.N_Floors]bool {
// 	var result [elevio.N_Floors][2]bool

// 	for i := 0; i < len(ActiveElevators); i++ {
// 		for floor := 0; floor < elevio.N_Floors; floor++ {
// 			result[floor][0] = result[floor][0] || ActiveElevators[i].Requests[floor][0]
// 			result[floor][1] = result[floor][1] || ActiveElevators[i].Requests[floor][1]
// 		}
// 	}
// }
