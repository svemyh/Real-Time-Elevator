package elevator

import (
	"elevator/elevio"
)

type ElevatorBehaviour int

const (
	EB_Idle ElevatorBehaviour = iota
	EB_DoorOpen
	EB_Moving
)

type ClearRequestVariant int

const (
	CV_All ClearRequestVariant = iota
	CV_InDoorn
)

type Config struct {
	ClearRequestVariant ClearRequestVariant
	DoorOpenDurationS   float64
}

type Elevator struct { // add local ip mby?
	Floor     int
	Dirn      elevio.Dirn
	Requests  [elevio.N_Floors][elevio.N_Buttons]bool
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
