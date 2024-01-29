package elevator

import (
	"elevator/elevio"
)

type ElevatorBehaviour int

const (
	EB_Idle     ElevatorBehaviour = 0
	EB_DoorOpen ElevatorBehaviour = 1
	EB_Moving   ElevatorBehaviour = 2
)

type ClearRequestVariant int

const (
	CV_All     ClearRequestVariant = 0
	CV_InDoorn ClearRequestVariant = 1
)

type Config struct {
	ClearRequestVariant ClearRequestVariant
	DoorOpenDurationS   float64
}

type Elevator struct {
	Floor     int
	Dirn      elevio.Dirn
	Requests  [elevio.N_Floors][elevio.N_Buttons]int
	Behaviour ElevatorBehaviour
	Config    Config
}

func ElevatorInit() Elevator {
	return Elevator{
		Floor:     -1,
		Dirn:      elevio.D_Stop,
		Behaviour: EB_Idle,
		Config: Config{
			ClearRequestVariant: CV_All,
			DoorOpenDurationS:   3.0,
		},
	}
}
