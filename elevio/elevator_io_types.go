package elevio

// Dirn represents the elevator's direction.
type Dirn int

// Enumeration for elevator directions.
const (
	D_Down Dirn = -1
	D_Stop Dirn = 0
	D_Up   Dirn = 1
)

// Button represents the types of buttons in the elevator.
type Button int

// Enumeration for elevator button types.
const (
	B_HallUp Button = iota
	B_HallDown
	B_Cab
)

// Constants for the number of floors and buttons.
const (
	N_Floors  = 4
	N_Buttons = 3
)

// ElevInputDevice represents the structure for elevator input devices.
type ElevInputDevice struct {
	FloorSensor   func() int
	RequestButton func(int, Button) int
	StopButton    func() int
	Obstruction   func() int
}

// ElevOutputDevice represents the structure for elevator output devices.
type ElevOutputDevice struct {
	FloorIndicator     func(int)
	RequestButtonLight func(int, Button, int)
	DoorLight          func(int)
	StopButtonLight    func(int)
	MotorDirection     func(Dirn)
}
