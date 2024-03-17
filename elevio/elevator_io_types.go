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

type ButtonType int

const (
	BT_HallUp   ButtonType = 0
	BT_HallDown ButtonType = 1
	BT_Cab      ButtonType = 2
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
