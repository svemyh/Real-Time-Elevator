package network

import (
	"elevator/elevator"
	"elevator/elevio"
	"elevator/hall_request_assigner"
	"time"
)

const (
	DETECTION_PORT string = ":14272"
	TCP_LISTEN_PORT string = ":14273"
	HALL_LIGHTS_PORT string = ":14274"
	TCP_BACKUP_PORT string = ":14275"
	TCP_NEW_PRIMARY_LISTEN_PORT = ":14276"
)

const bufSize = 1024
const udpInterval = 500 * time.Millisecond

type MessageType string

const (
	TypeActiveElevator 		 MessageType = "ActiveElevator"
	TypeButtonEvent    		 MessageType = "ButtonEvent"
	TypeACK            		 MessageType = "ACK"
	TypeString         		 MessageType = "string"
	TypeCombinedHallRequests MessageType = "CombinedHallRequests"
)

type Message interface{}

type MsgActiveElevator struct {
	Type    MessageType                          		`json:"type"`
	Content hall_request_assigner.ActiveElevator 		"json:content"
}

type MsgButtonEvent struct {
	Type    MessageType        							`json:"type"`
	Content elevio.ButtonEvent 							"json:content" // refactor: change Content to antother name? Then go compiler stops complaining
}

type MsgACK struct {
	Type    MessageType 								`json:"type"`
	Content bool        								"json:content"
}

type MsgString struct {
	Type    MessageType 								`json:"type"`
	Content string      								"json:content"
}

type MsgCombinedHallRequests struct {
	Type    MessageType 								`json:"type"`
	Content [elevio.N_Floors][elevio.N_Buttons - 1]bool  "json:content"
}

type ClientUpdate struct {
	Client []string
	New    string
	Lost   []string
}

type ElevatorSystemChannels struct { 										// Find a better name. It got ONE map smh
	ActiveElevatorMap 		  map[string]elevator.Elevator
	CombinedHallRequests 	  [elevio.N_Floors][elevio.N_Buttons-1]bool		// As this asks for HallRequests, one row of buttons composing CabRequests gets subtracted
	StateUpdateCh             chan hall_request_assigner.ActiveElevator
	HallOrderCompleteCh       chan elevio.ButtonEvent
	DisconnectedElevatorCh    chan string
	AssignHallRequestsMapCh   chan map[string][elevio.N_Floors][elevio.N_Buttons - 1]bool
	AckCh                     chan bool
}

type FSMSystemChannels struct {
	FSMStateUpdateCh          chan hall_request_assigner.ActiveElevator
	FSMHallOrderCompleteCh    chan elevio.ButtonEvent
	FSMAssignedHallRequestsCh chan [elevio.N_Floors][elevio.N_Buttons - 1]bool
}
