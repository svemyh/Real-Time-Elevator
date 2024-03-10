package elevio


// Incomplete
// func ElevGetInputDevice() ElevInputDevice {

// 	floorSensorCh := make(chan int)
// 	requestButtonCh := make(chan ButtonEvent)
// 	stopButtonCh := make(chan bool)
// 	obstructionSwitchCh := make(chan bool)

// 	return ElevInputDevice{
// 		go PollFloorSensor(floorSensorCh)
// 		go PollButtons(requestButtonCh)
// 		go PollStopButton(stopButtonCh)
// 		go PollObstructionSwitch(obstructionSwitchCh)
// 		FloorSensor := <-floorSensorCh
// 		RequestButton := <-requestButtonCh
// 		StopButton := <-stopButtonCh
// 		Obstruction := <-obstructionSwitchCh
// 	}
// }

// // Incomplete
// func ElevGetOutputDevice() ElevOutputDevice {
// 	ElevSetFloorIndicator(floor int)

// 	return ElevOutputDevice{

// 		RequestButtonLight: ElevSetButtonLight(button ButtonType, floor int, value bool),
// 		DoorLight:          ElevSetDoorOpenLamp(value bool),
// 		StopButtonLight:    ElevSetStopLamp(value bool),
// 		MotorDirection:     ElevSetMotorDirection(dirn Dirn),
// 	}
// }

func NewElevInputDevice() ElevInputDevice {
	device := ElevInputDevice{
		FloorSensorCh:   make(chan int),
		RequestButtonCh: make(chan ButtonEvent),
		StopButtonCh:    make(chan bool),
		ObstructionCh:   make(chan bool),
	}

	go PollFloorSensor(device.FloorSensorCh)
	go PollButtons(device.RequestButtonCh)
	go PollStopButton(device.StopButtonCh)
	go PollObstructionSwitch(device.ObstructionCh)

	return device
}

func DirnToString(d Dirn) string {
	switch d {
	case D_Up:
		return "D_Up"
	case D_Down:
		return "D_Down"
	case D_Stop:
		return "D_Stop"
	default:
		return "D_UNDEFINED"
	}
}

func ButtonToString(b Button) string {
	switch b {
	case B_HallUp:
		return "B_HallUp"
	case B_HallDown:
		return "B_HallDown"
	case B_Cab:
		return "B_Cab"
	default:
		return "B_UNDEFINED"
	}
}


