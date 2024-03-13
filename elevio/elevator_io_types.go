package elevio

const (
	N_Floors = 4
	N_Buttons = 3
)

type Dirn int

const (
	D_Down Dirn = -1
	D_Stop Dirn = 0
	D_Up   Dirn = 1
)

type Button int

const (
	B_HallUp   Button = 0
	B_HallDown Button = 1
	B_Cab      Button = 2
)

type ButtonEvent struct {
	Floor  int
	Button ButtonType
}

type ElevInputDevice struct {
	FloorSensorCh   chan int
	RequestButtonCh chan ButtonEvent
	StopButtonCh    chan bool
	ObstructionCh   chan bool
}


