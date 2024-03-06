package main

import (
	"elevator/elevator"
	"elevator/elevio"
	"elevator/hall_request_assigner"
	"fmt"
)

func main() {
	elev1 := elevator.Elevator{
		Floor:     1,
		Dirn:      elevio.D_Down,                                                                                                                  // Assuming elevio.DirnUp is a valid constant
		Requests:  [elevio.N_Floors][elevio.N_Buttons]bool{{true, true, false}, {true, true, false}, {true, false, false}, {false, false, false}}, // Example, adjust based on actual N_Floors and N_Buttons
		Behaviour: elevator.EB_Idle,                                                                                                               // Replace someBehaviourValue with actual value
		Config: elevator.Config{
			ClearRequestVariant: elevator.CV_All,
			DoorOpenDurationS:   3.0,
		},
	}

	elev2 := elevator.Elevator{
		Floor:     1,
		Dirn:      elevio.D_Down,                                                                                                                 // Assuming elevio.DirnUp is a valid constant
		Requests:  [elevio.N_Floors][elevio.N_Buttons]bool{{true, true, false}, {true, true, false}, {true, false, true}, {false, false, false}}, // Example, adjust based on actual N_Floors and N_Buttons
		Behaviour: elevator.EB_Idle,                                                                                                              // Replace someBehaviourValue with actual value
		Config: elevator.Config{
			ClearRequestVariant: elevator.CV_All,
			DoorOpenDurationS:   3.0,
		},
	}

	activElev1 := hall_request_assigner.ActiveElevator{Elevator: elev1, MyAddress: "555"}
	activElev2 := hall_request_assigner.ActiveElevator{Elevator: elev2, MyAddress: "666"}

	var activeElevators []hall_request_assigner.ActiveElevator
	activeElevators = append(activeElevators, activElev1, activElev2)

	e := hall_request_assigner.RequestsToCab(elev1.Requests)
	println(e)

	HRAStates := hall_request_assigner.ActiveElevatorsToHRAElevatorState(activElev1)
	fmt.Printf("HRAStates: %v\n", HRAStates)
}
