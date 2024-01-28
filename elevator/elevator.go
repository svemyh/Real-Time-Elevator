package elevator

import (
	"elevator/elevio"
)

type ElevatorBehaviour struct {
	EB_Idle = 0
	EB_DoorOpen = 1
	EB_Moving = 2
}

type ClearRequestVariant struct {
	CV_All = 0
	CV_InDoorn = 1
}


type Elevator struct {
	Floor int
	Dirn Dirn
	Requests [N_FLOORS][N_BUTTONS]int
	Behaviour ElevatorBehaviour 

	Config struct {
		ClearRequestVariant ClearRequestVariant
		DoorOpenDurationS   double
	}
}


func ElevatorInit() Elevator {
	return Elevator{
		Floor: -1,
		Dirn: D_Stop,
		Behaviour: EB_Idle,
		Config: {
			ClearRequestVariant: CV_All,
			DoorOpenDurationS: 3.0,
		},
	}
}
