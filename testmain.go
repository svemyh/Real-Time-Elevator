package main

import (
	"elevator/hall_request_assigner"
	"fmt"
)

func main() {
	// The following code demonstrates the hall_request_assigner functionality.

	activeElev := hall_request_assigner.InitActiveElevator() // Initializes a default elevator with floor 1, direction stop, and behavior idle. This is the elevator that will be used to test the hall_request_assigner functionality.

	var ActiveElevators []hall_request_assigner.ActiveElevator // Initializes ActiveElevators.
	ActiveElevators = append(ActiveElevators, activeElev)      // Adds an elevator to ActiveElevators.

	// The following []ActiveElevator object contains updated hall requests determined by hall_request_assigner.exe. Not certain if cab-requests should or should not be updated aswell. Question: Should we clear the cab requests here? - Answer depends on how hall_request_assigner.exe is coded. TODO: Check functionality by enabling/disabling this at later time when functioning elevators are acheived.
	NewActiveElevators := hall_request_assigner.HallRequestAssigner(ActiveElevators) // Computes new hall requests for the elevators in ActiveElevators. Returns object of same type as ActiveElevators.
	fmt.Println("NewActiveElevators:", NewActiveElevators)

}
