package network

import (
	"elevator/elevator"
	"elevator/elevio"
	"elevator/hall_request_assigner"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

var DETECTION_PORT string = ":14272"
var TCP_LISTEN_PORT string = ":14279"
var HALL_LIGHTS_PORT string = ":14274"
var TCP_BACKUP_PORT string = ":14275"
var TCP_NEW_PRIMARY_LISTEN_PORT string = ":14276"

type MessageType string

const bufSize = 1024

const udpInterval = 2 * time.Second
const timeout = 500 * time.Millisecond

const (
	TypeActiveElevator       MessageType = "ActiveElevator"
	TypeButtonEvent          MessageType = "ButtonEvent"
	TypeACK                  MessageType = "ACK"
	TypeString               MessageType = "string"
	TypeCombinedHallRequests MessageType = "CombinedHallRequests"
	TypePing                 MessageType = "PING"
	TypeTimestamp            MessageType = "Timestamp"
)

type Message interface{}

type MsgActiveElevator struct {
	Type    MessageType                          `json:"type"`
	Content hall_request_assigner.ActiveElevator "json:content"
}

type MsgButtonEvent struct {
	Type    MessageType        `json:"type"`
	Content elevio.ButtonEvent "json:content" // refactor: change Content to antother name? Then go compiler stops complaining
}

type MsgACK struct { // TODO: Refactor to simply MsgString - is used in two or more different contexts
	Type    MessageType `json:"type"`
	Content bool        "json:content"
}

type MsgString struct {
	Type    MessageType `json:"type"`
	Content string      "json:content"
}

type MsgCombinedHallRequests struct { // TODO: Refactor to simply MsgHallRequests - is used in two or more different contexts
	Type    MessageType                                 `json:"type"`
	Content [elevio.N_Floors][elevio.N_Buttons - 1]bool "json:content"
}

type MsgPing struct { // TODO: Refactor to simply MsgString - is used in two or more different contexts
	Type    MessageType `json:"type"`
	Content string      "json:content"
}

type MsgTimestamp struct { // TODO: Refactor to simply MsgString - is used in two or more different contexts
	Type    MessageType `json:"type"`
	Content string      "json:content"
}

type ClientUpdate struct {
	Client []string
	New    string
	Lost   []string
}

type ElevatorSystemChannels struct {
	FSMStateUpdateCh          chan hall_request_assigner.ActiveElevator
	FSMHallOrderCompleteCh    chan elevio.ButtonEvent
	StateUpdateCh             chan hall_request_assigner.ActiveElevator
	HallOrderCompleteCh       chan elevio.ButtonEvent
	DisconnectedElevatorCh    chan string
	FSMAssignedHallRequestsCh chan [elevio.N_Floors][elevio.N_Buttons - 1]bool
	AssignHallRequestsMapCh   chan map[string][elevio.N_Floors][elevio.N_Buttons - 1]bool
	AckCh                     chan bool
}

func NewElevatorSystemChannels() ElevatorSystemChannels {
	return ElevatorSystemChannels{
		FSMStateUpdateCh:          make(chan hall_request_assigner.ActiveElevator, 1024),
		FSMHallOrderCompleteCh:    make(chan elevio.ButtonEvent, 1024),
		StateUpdateCh:             make(chan hall_request_assigner.ActiveElevator, 1024),
		HallOrderCompleteCh:       make(chan elevio.ButtonEvent, 1024),
		DisconnectedElevatorCh:    make(chan string, 1024),
		FSMAssignedHallRequestsCh: make(chan [elevio.N_Floors][elevio.N_Buttons - 1]bool, 1024),
		AssignHallRequestsMapCh:   make(chan map[string][elevio.N_Floors][elevio.N_Buttons - 1]bool, 1024),
		AckCh:                     make(chan bool, 64),
	}
}

// Alias: RunPrimaryBackup()
func InitNetwork(FSMStateUpdateCh chan hall_request_assigner.ActiveElevator, FSMHallOrderCompleteCh chan elevio.ButtonEvent, StateUpdateCh chan hall_request_assigner.ActiveElevator, HallOrderCompleteCh chan elevio.ButtonEvent, DisconnectedElevatorCh chan string, FSMAssignedHallRequestsCh chan [elevio.N_Floors][elevio.N_Buttons - 1]bool, AssignHallRequestsCh chan map[string][elevio.N_Floors][elevio.N_Buttons - 1]bool, AckCh chan bool) {
	clientUpdateCh := make(chan ClientUpdate)
	//clientTxEnable := make(chan bool)
	var id string

	isPrimary, primaryAddress := AmIPrimary(DETECTION_PORT, clientUpdateCh)
	if isPrimary {
		if id == "" {
			localIP, err := LocalIP()
			if err != nil {
				fmt.Println(err)
				localIP = "DISCONNECTED"
			}
			id = fmt.Sprintf("Master-%s-%d", localIP, os.Getpid())
			fmt.Printf("My id: %s\n", id)
		}

		log.Println("Operating as primary...")

		//init empty activeElevMap and CombinedHallReq
		var CombinedHallRequests [elevio.N_Floors][elevio.N_Buttons - 1]bool
		ActiveElevatorMap := make(map[string]elevator.Elevator)
		ConsumerChannels := make(map[net.Conn]chan map[string][elevio.N_Floors][elevio.N_Buttons - 1]bool)
		go PrimaryRoutine(ActiveElevatorMap, CombinedHallRequests, StateUpdateCh, HallOrderCompleteCh, DisconnectedElevatorCh, AssignHallRequestsCh, AckCh, ConsumerChannels)
		time.Sleep(1500 * time.Millisecond)
		TCPDialPrimary(GetLocalIPv4()+TCP_LISTEN_PORT, FSMStateUpdateCh, FSMHallOrderCompleteCh, FSMAssignedHallRequestsCh)
	} else {
		if id == "" {
			localIP, err := LocalIP()
			if err != nil {
				fmt.Println(err)
				localIP = "DISCONNECTED"
			}
			id = fmt.Sprintf("Client-%s-%d\n", localIP, os.Getpid())
			fmt.Printf("My id: %s", id)
		}
		log.Println("Operating as client...")
		go TCPDialPrimary(primaryAddress+TCP_LISTEN_PORT, FSMStateUpdateCh, FSMHallOrderCompleteCh, FSMAssignedHallRequestsCh)
		go TCPListenForNewPrimary(TCP_NEW_PRIMARY_LISTEN_PORT, FSMStateUpdateCh, FSMHallOrderCompleteCh, FSMAssignedHallRequestsCh)
		conn, err := TCPListenForBackupPromotion(TCP_BACKUP_PORT) //will simply be a net.Listen("TCP", "primaryAdder"). This blocks code until a connection is established
		if err != nil {
			panic(err)
		}
		BackupRoutine(conn, primaryAddress+DETECTION_PORT, StateUpdateCh, HallOrderCompleteCh, DisconnectedElevatorCh, AssignHallRequestsCh, AckCh)
	}
}

// Checks the event that a backup has become a new primary and wants to establish connection. This go routine should be shut down at some point
func TCPListenForNewPrimary(TCPPort string, FSMStateUpdateCh chan hall_request_assigner.ActiveElevator, FSMHallOrderCompleteCh chan elevio.ButtonEvent, FSMAssignedHallRequestsCh chan [elevio.N_Floors][elevio.N_Buttons - 1]bool) {
	fmt.Println("- Executing TCPListenForNewPrimary()")
	//listen for new elevators on TCP port
	//when connection established run the go routine TCPReadElevatorStates to start reading data from the conn
	//go TCPReadElevatorStates(stateUpdateCh)
	ls, err := net.Listen("tcp", TCPPort)
	if err != nil {
		fmt.Println("The connection failed. Error:", err)
		return
	}
	defer ls.Close()

	fmt.Println("-TCPListenForNewPrimary() listening for new primary connections to port:", TCPPort)
	for {
		conn, err := ls.Accept()
		if err != nil {
			fmt.Println("Error: ", err)
			continue
		}

		//go RecieveAssignedHallRequests(conn, FSMAssignedHallRequestsCh)                 // REPLACE TO: TCPReadAssignedHallRequests()
		go TCPReadAssignedHallRequests(conn, FSMAssignedHallRequestsCh)
		go sendLocalStatesToPrimaryLoop(conn, FSMStateUpdateCh, FSMHallOrderCompleteCh) // This will terminate whenever the connection/conn is closed - i.e. conn.Write() throws an error.
	}
}

// will simply be a net.Listen("TCP", "TCP_BACKUP_PORT"). This blocks code until a connection is established
func TCPListenForBackupPromotion(port string) (net.Conn, error) {
	fmt.Println(" - Executing TCPListenForBackupPromotion()")

	ls, err := net.Listen("tcp", port)
	if err != nil {
		fmt.Println("TCPListenForBackupPromotion - The connection failed. Error:", err)
		return nil, err
	}
	defer ls.Close()

	fmt.Println("TCPListenForBackupPromotion -  listening for new backup connections to port:", port)
	for {
		conn, err := ls.Accept()
		if err != nil {
			fmt.Println("Error: ", err)
			continue
		}
		return conn, nil
	}
}

func TCPDialPrimary(PrimaryAddress string, FSMStateUpdateCh chan hall_request_assigner.ActiveElevator, FSMHallOrderCompleteCh chan elevio.ButtonEvent, FSMAssignedHallRequestsCh chan [elevio.N_Floors][elevio.N_Buttons - 1]bool) {
	fmt.Println("Connecting by TCP to the address: ", PrimaryAddress)

	conn, err := net.Dial("tcp", PrimaryAddress)
	if err != nil {
		fmt.Println("Connection failed. Error: ", err)
		return
	}

	fmt.Println("Conection established to: ", conn.RemoteAddr())
	defer conn.Close()

	// go RecieveAssignedHallRequests(conn, FSMAssignedHallRequestsCh) // REPLACE TO: TCPReadAssignedHallRequests()
	go TCPReadAssignedHallRequests(conn, FSMAssignedHallRequestsCh)
	sendLocalStatesToPrimaryLoop(conn, FSMStateUpdateCh, FSMHallOrderCompleteCh)
}

func RecieveAssignedHallRequests(conn net.Conn, FSMAssignedHallRequestsCh chan [elevio.N_Floors][elevio.N_Buttons - 1]bool) { // NOT TESTED!
	// NB: To be replaced by TCPReadAssignedHallRequests()
	fmt.Printf("RecieveAssignedHallRequests() - *New connection accepted from address: %s to %s\n", conn.LocalAddr(), conn.RemoteAddr().String())

	defer conn.Close()

	for {
		var buf [bufSize]byte
		n, err := conn.Read(buf[:])
		if err != nil {
			// Error means TCP-conn has broken -> TODO: Do something
			println("OH NO, The conn at line 224 broke1: %e", err)
			//panic(err)
			break
		}

		// Unmarshal JSON data into a map of elevator states
		var assignedHallRequests [elevio.N_Floors][elevio.N_Buttons - 1]bool
		err = json.Unmarshal(buf[:n], &assignedHallRequests)
		if err != nil {
			//return err
			println("OH NO, The conn at line 224 broke2: %e", err)
			//panic(err)
			break
		}
		FSMAssignedHallRequestsCh <- assignedHallRequests
		fmt.Println("FSMAssignedHallRequestsCh <- assignedHallRequests & conn.LocalAddr(): ", assignedHallRequests, conn.LocalAddr())
		time.Sleep(50 * time.Millisecond)
	}
}

// Recieves assigned hall orders from cost-function and feeds it into FSM of local machine. Message is recieved from TCPWriteAssignedHallRequests().
func TCPReadAssignedHallRequests(conn net.Conn, FSMAssignedHallRequestsCh chan [elevio.N_Floors][elevio.N_Buttons - 1]bool) {
	// NB: Replaces RecieveAssignedHallRequests()
	fmt.Printf("TCPReadAssignedHallRequests() - *New connection accepted from address: %s to %s\n", conn.LocalAddr(), conn.RemoteAddr().String())

	defer conn.Close()
	for {
		time.Sleep(50 * time.Millisecond)
		//// ------------------------------------------------------------------

		// Create buffer and read data into the buffer using conn.Read()
		var buf [bufSize]byte
		n, err := conn.Read(buf[:])
		if err != nil {
			// Error means TCP-conn has broken -> Need to feed this signal to drop the conn's respective ActiveElevator from Primary's ActiveElevators. It is now considered inactive.
			//DisconnectedElevatorCh <- conn.RemoteAddr().String() // Question: Should this be LocalAddr() or RemoteAddr() or both?
			break
			//conn.Close()
			//panic(err)
		}

		// Decoding said data into a json-style object
		var genericMsg map[string]interface{}
		if err := json.Unmarshal(buf[:n], &genericMsg); err != nil {
			fmt.Println("Error unmarshaling generic message: ", err)
			panic(err)
		}
		// Based on MessageType (which is an element of each struct sent over connection) determine how its corresponding data should be decoded.
		switch MessageType(genericMsg["type"].(string)) {
		case TypeCombinedHallRequests:
			var msg MsgCombinedHallRequests
			if err := json.Unmarshal(buf[:n], &msg); err != nil {
				panic(err) //TODO: can be changed to continue, but has as panic for debug purpuses
			}
			fmt.Printf("Received string object: %+v\n", msg)
			FSMAssignedHallRequestsCh <- msg.Content
			fmt.Println("FSMAssignedHallRequestsCh <- msg.Content & conn.LocalAddr(): ", msg.Content, conn.LocalAddr())
		case TypePing:
			var msg MsgPing
			if err := json.Unmarshal(buf[:n], &msg); err != nil {
				panic(err) //TODO: can be changed to continue, but has as panic for debug purpuses
			}
			fmt.Println("TCPReadAssignedHallRequests() - Recieved PING: ", msg.Content, " -RemoteAddr: ", conn.LocalAddr().String(), "-RemoteAddr: ", conn.RemoteAddr().String())
		case TypeTimestamp:
			var msg MsgString
			if err := json.Unmarshal(buf[:n], &msg); err != nil {
				panic(err) //TODO: can be changed to continue, but has as panic for debug purpuses
			}
			fmt.Println("TCPReadAssignedHallRequests() - Recieved TypeTimestamp: ", msg.Content, " -RemoteAddr: ", conn.LocalAddr().String(), "-RemoteAddr: ", conn.RemoteAddr().String())
			TimestampMsg := MsgString{
				Type:    TypeTimestamp,
				Content: msg.Content,
			}
			data, err := json.Marshal(TimestampMsg)
			if err != nil {
				fmt.Printf("Failed to encode MsgString to json: %v\n", err)
			}
			_, err = conn.Write(data)
			if err != nil {
				fmt.Printf("Failed to return heartbeat: %v\n", err)
			}
			fmt.Println(" -- SendHeartbeats sent ", TimestampMsg, "from localaddr:", conn.LocalAddr().String(), "to remoteaddr: ", conn.RemoteAddr().String())
			time.Sleep(350 * time.Millisecond) // Try reducing this to minimal possible value.
		default:
			fmt.Println("TCPReadAssignedHallRequests() - Unknown message type")
		}
	}
}

func sendLocalStatesToPrimaryLoop(conn net.Conn, FSMStateUpdateCh chan hall_request_assigner.ActiveElevator, FSMHallOrderCompleteCh chan elevio.ButtonEvent) {
	fmt.Printf("sendLocalStatesToPrimaryLoop() - *New connection accepted from address: %s to %s\n", conn.LocalAddr(), conn.RemoteAddr().String())

	defer conn.Close()
	for {
		select {
		case stateUpdate := <-FSMStateUpdateCh:
			/*
				my_ActiveElevatorMsg := MsgActiveElevator{
					Type:    TypeActiveElevator,
					Content: stateUpdate,
				}

				data, err := json.Marshal(my_ActiveElevatorMsg)
				if err != nil {
					fmt.Println("Error sending to Primary: ", err)
					return
				}

				_, err = conn.Write(data)
				if err != nil {
					fmt.Println("Error sending to Primary: ", err)
					return
				}
				time.Sleep(50 * time.Millisecond)
			*/
			TCPSendActiveElevator(conn, stateUpdate)

		case hallOrderComplete := <-FSMHallOrderCompleteCh:
			my_ButtonEventMsg := MsgButtonEvent{
				Type:    TypeButtonEvent,
				Content: hallOrderComplete,
			}

			data, err := json.Marshal(my_ButtonEventMsg)
			if err != nil {
				fmt.Println("Error encoding MsgButtonEvent to json: ", err)
				return
			}

			fmt.Println("Writing a ButtonEvent to the Primary:", hallOrderComplete)
			_, err = conn.Write(data)
			if err != nil {
				fmt.Println("Error sending to Primary: ", err)
				return
			}
			time.Sleep(50 * time.Millisecond)
		}
		time.Sleep(50 * time.Millisecond)
	}
}

//receiverChan := make(chan string)
//go network.Reciever(receiverChan, "localhost:20013")

// Sends to RecieveAssignedHallRequests
func TCPWriteElevatorStates(conn net.Conn, personalAssignedHallRequestsCh chan map[string][elevio.N_Floors][elevio.N_Buttons - 1]bool) {
	// NB: To be replaced by TCPWriteAssignedHallRequests()
	defer conn.Close()

	for {
		assignedHallRequests := <-personalAssignedHallRequestsCh
		data, err := json.Marshal(assignedHallRequests[conn.RemoteAddr().(*net.TCPAddr).IP.String()])
		if err != nil {
			fmt.Println("Error encoding hallRequests to json: ", err)
			return
		}

		_, err = conn.Write(data)
		if err != nil {
			fmt.Println("Error sending HallRequests to: ", err)
			return
		}
		time.Sleep(500 * time.Millisecond)
		log.Print("Current time")
		fmt.Println("assignedHallRequests total -- ", assignedHallRequests)
		fmt.Println("assigned hall req to thsi conn:: ", assignedHallRequests[conn.RemoteAddr().(*net.TCPAddr).IP.String()])
		fmt.Println("conn that is sending: ", conn.RemoteAddr().String())
		fmt.Println("How we format the conn: ", conn.RemoteAddr().(*net.TCPAddr).IP.String())
	}
}

// Distributes assigned hall requests out to the individual FSMs. Sends to TCPReadAssignedHallRequests().
func TCPWriteAssignedHallRequests(conn net.Conn, personalAssignedHallRequestsCh chan map[string][elevio.N_Floors][elevio.N_Buttons - 1]bool, DisconnectedElevatorCh chan string, ReadHeartbeatsCh chan string) {
	defer conn.Close()
	// NB: Replaces TCPWriteElevatorStates
	// TODO: Package content into json of type MsgCombinedElevatorRequests before sending
	// TODO: Add a go-routine for checking if conn is alive by sending PINGs
	errCh := make(chan error)

	go SendHeartbeats(conn, errCh, ReadHeartbeatsCh)
	for {
		select {
		case assignedHallRequests := <-personalAssignedHallRequestsCh:
			myAssignedHallRequestsMsg := MsgCombinedHallRequests{
				Type:    TypeActiveElevator,
				Content: assignedHallRequests[conn.RemoteAddr().(*net.TCPAddr).IP.String()],
			}
			fmt.Println("myAssignedHallRequestsMsg:", myAssignedHallRequestsMsg)
			data, err := json.Marshal(myAssignedHallRequestsMsg)
			if err != nil {
				fmt.Println("Error encoding AssignedHallRequests to json: ", err)
				return
			}

			_, err = conn.Write(data)
			if err != nil {
				fmt.Println("Error sending AssignedHallRequestsMsg: ", err)
				return
			}
			time.Sleep(50 * time.Millisecond)
			log.Print("CURRENT TIME")
			fmt.Println("assignedHallRequests total -- ", assignedHallRequests)
			fmt.Println("assigned hall req to thsi conn: ", assignedHallRequests[conn.RemoteAddr().(*net.TCPAddr).IP.String()])
			fmt.Println("conn that is sending: ", conn.RemoteAddr().String())
			fmt.Println("How we format the conn: ", conn.RemoteAddr().(*net.TCPAddr).IP.String())
		case err := <-errCh:
			fmt.Printf("Connection error, terminating TCPWriteAssignedHallRequests: %v\n", err)
			DisconnectedElevatorCh <- conn.RemoteAddr().(*net.TCPAddr).IP.String() // TODO: Test this
			return

		}
	}
}

// UNUSED
// Continously monitors that a net.Conn is still alive
func SendHeartbeatsV0(conn net.Conn, errCh chan<- error, ReadHeartbeatsCh chan string) {
	timestamp := time.Now().Format("15:04:05")
	timestampCh := make(chan string, 1024)
	timestampCh <- timestamp
	fmt.Println("INIT SendHeartbeats() -  timestamp:", timestamp)

	go func(timestampCh chan string, errCh chan<- error) {
		for {
			select {
			case t := <-timestampCh:
				fmt.Println("t := <-timestampCh recieved: ", t)
				timestamp = t
				time.Sleep(750 * time.Millisecond)
			default:
				TimestampMsg := MsgString{
					Type:    TypeTimestamp,
					Content: timestamp,
				}
				data, err := json.Marshal(TimestampMsg)
				if err != nil {
					fmt.Printf("Failed to encode MsgString to json: %v\n", err)
					errCh <- err
					return
				}
				_, err = conn.Write(data)
				if err != nil {
					fmt.Printf("Failed to send heartbeat: %v\n", err)
					errCh <- err
					return
				}
				fmt.Println(" -- SendHeartbeats sent ", TimestampMsg, "from localaddr:", conn.LocalAddr().String(), "to remoteaddr: ", conn.RemoteAddr().String())
				time.Sleep(750 * time.Millisecond) // Try reducing this to minimal possible value.
			}
		}
	}(timestampCh, errCh)

	for {
		select {
		case t := <-ReadHeartbeatsCh:
			if t == timestamp {
				fmt.Println("Recieved timestamp as ACK: ", t)
				timestamp = time.Now().Format("15:04:05")
				timestampCh <- timestamp
			}

		case <-time.After(3 * time.Second):
			fmt.Println("Heartbeat not acknowledged in time.")
			errCh <- errors.New("heartbeat timeout: connection might be broken")
			close(timestampCh)
			return
		}
	}
}

// Continously monitors that a net.Conn is still alive
func SendHeartbeats(conn net.Conn, errCh chan<- error, ReadHeartbeatsCh chan string) {
	timestampCh := make(chan string, 1024)

	timestamp := time.Now().Format("15:04:05")
	timestampCh <- timestamp

	fmt.Println("INIT SendHeartbeats() -  timestamp:", timestamp)
	for {
		select {
		case t := <-ReadHeartbeatsCh:
			if t == timestamp {
				fmt.Println("Recieved timestamp as ACK: ", t)
				timestamp = time.Now().Format("15:04:05")
				timestampCh <- timestamp
			}
		case tt := <-timestampCh:
			fmt.Println("tt := <-timestampCh recieved: ", tt)
			timestamp = tt
			time.Sleep(750 * time.Millisecond)

		case <-time.After(5 * time.Second):
			fmt.Println("____-----_____-----_____---- Heartbeat not acknowledged in time.")
			errCh <- errors.New("heartbeat timeout: connection might be broken")
			close(timestampCh)
			return

		default:
			TimestampMsg := MsgString{
				Type:    TypeTimestamp,
				Content: timestamp,
			}
			data, err := json.Marshal(TimestampMsg)
			if err != nil {
				fmt.Printf("Failed to encode MsgString to json: %v\n", err)
				errCh <- err
				return
			}
			_, err = conn.Write(data)
			if err != nil {
				fmt.Printf("Failed to send heartbeat: %v\n", err)
				errCh <- err
				return
			}
			fmt.Println(" -- SendHeartbeats sent ", TimestampMsg, "from localaddr:", conn.LocalAddr().String(), "to remoteaddr: ", conn.RemoteAddr().String())
			time.Sleep(350 * time.Millisecond) // Try reducing this to minimal possible value.
		}
	}
}

// Recieves message on AssignedHallRequestsCh and distributes said message to all consumer go-routines ConsumerAssignedHallRequestsCh
func StartBroadcaster(AssignedHallRequestsCh chan map[string][elevio.N_Floors][elevio.N_Buttons - 1]bool, Consumers map[net.Conn]chan map[string][elevio.N_Floors][elevio.N_Buttons - 1]bool) {
	for hallRequests := range AssignedHallRequestsCh {
		for _, ch := range Consumers {
			ch <- hallRequests
		}
	}
}

// Alias: Server()
func TCPReadElevatorStates(conn net.Conn, StateUpdateCh chan hall_request_assigner.ActiveElevator, HallOrderCompleteCh chan elevio.ButtonEvent, DisconnectedElevatorCh chan string, ReadHeartbeatsCh chan string) {
	//TODO:Read the states and store in a buffer
	//TODO: Check if the read data was due to local elevator reaching a floor and clearing a request (send cleared request on OrderCompleteCh)
	//TODO:send the updated states on stateUpdateCh so that it can be read in HandlePrimaryTasks(StateUpdateCh)
	// Can be added/expanded: LocalErrorDetectedCh or similar
	// type StateUpdateCh = IP + elevatorStates
	// type HallOrderCopleteCh = floor number (of cab call completed)

	fmt.Printf("TCPReadElevatorStates() - *New connection accepted from address: %s\n", conn.LocalAddr())
	fmt.Printf("TCPReadElevatorStates() - *New connection accepted from address: %s to %s\n", conn.LocalAddr(), conn.RemoteAddr().String())

	defer conn.Close()

	for {
		// Create buffer and read data into the buffer using conn.Read()
		var buf [bufSize]byte
		n, err := conn.Read(buf[:])
		if err != nil {
			// Error means TCP-conn has broken -> Need to feed this signal to drop the conn's respective ActiveElevator from Primary's ActiveElevators. It is now considered inactive.
			DisconnectedElevatorCh <- conn.RemoteAddr().String() // Question: Should this be LocalAddr() or RemoteAddr() or both?
			break
			//conn.Close()
			//panic(err)
		}

		// Decoding said data into a json-style object
		var genericMsg map[string]interface{}
		if err := json.Unmarshal(buf[:n], &genericMsg); err != nil {
			fmt.Println("Error unmarshaling generic message: ", err)
			panic(err)
		}
		// Based on MessageType (which is an element of each struct sent over connection) determine how its corresponding data should be decoded.
		switch MessageType(genericMsg["type"].(string)) {
		case TypeActiveElevator:
			var msg MsgActiveElevator
			if err := json.Unmarshal(buf[:n], &msg); err != nil {
				panic(err) //TODO: can be changed to continue, but has as panic for debug purpuses
			}
			fmt.Printf("Received ActiveElevator object: %+v\n", msg)
			StateUpdateCh <- msg.Content

		case TypeButtonEvent:
			var msg MsgButtonEvent
			if err := json.Unmarshal(buf[:n], &msg); err != nil {
				panic(err) //TODO: can be changed to continue, but has as panic for debug purpuses
			}
			fmt.Printf("Received ButtonEvent object: %+v\n", msg)
			HallOrderCompleteCh <- msg.Content
		case TypeString:
			var msg MsgString
			if err := json.Unmarshal(buf[:n], &msg); err != nil {
				panic(err) //TODO: can be changed to continue, but has as panic for debug purpuses
			}
			fmt.Printf("Received string object: %+v\n", msg)
			DisconnectedElevatorCh <- msg.Content
		case TypePing:
			var msg MsgPing
			if err := json.Unmarshal(buf[:n], &msg); err != nil {
				panic(err) //TODO: can be changed to continue, but has as panic for debug purpuses
			}
			fmt.Println("TCPReadElevatorStates() - Recieved PING: ", msg.Content, " -RemoteAddr: ", conn.LocalAddr().String(), "-RemoteAddr: ", conn.RemoteAddr().String())
		case TypeTimestamp:
			var msg MsgString
			if err := json.Unmarshal(buf[:n], &msg); err != nil {
				panic(err) //TODO: can be changed to continue, but has as panic for debug purpuses
			}
			fmt.Println("TCPReadElevatorStates() - Recieved TypeTimestamp: ", msg.Content, " -RemoteAddr: ", conn.LocalAddr().String(), "-RemoteAddr: ", conn.RemoteAddr().String())
			ReadHeartbeatsCh <- msg.Content
		default:
			fmt.Println("Unknown message type")
		}
	}
}

func TCPDialBackup(address string, port string) net.Conn {
	fmt.Println("TCPDialBackup() - Connecting by TCP to the address: ", address+port)

	conn, err := net.Dial("tcp", address+port)
	if err != nil {
		fmt.Println("Error in TCPDialBackup() - Connection failed. Error: ", err)
		return nil
	}

	fmt.Println("TCPDialBackup() - Conection established to: ", conn.RemoteAddr())
	return conn
}

// Can be used for testing purposes for writing either a ActiveElevator or ButtonEvent to TCPReadElevatorStates
func StartClient(port string, msg Message) {
	conn, err := net.Dial("tcp", "localhost"+port)
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	data, err := json.Marshal(msg)
	if err != nil {
		panic(err)
	}

	_, err = conn.Write(data)
	if err != nil {
		panic(err)
	}
}

/*
func TCPListenForNewPrimary() {
	//listen for new primary on tcp port and accept
}

func TCPListenForNewElevators(port string, listenerconnection, receiverchannels){
	//listen for new elevators on TCP port
	//when connection established run the go routine TCPReadElevatorStates to start reading data from the conn
	go run TCPReadElevatorStates(stateUpdateCh)

	allClients := make(map[net.Conn]string)
	newConnections := make(chan net.Conn)
	deadConnections := make(chan net.Conn)
	messages := make(chan connectionMsg)

	go acceptConnections(connection, newConnections)

	for {
		select {
		case conn := <-newConnections:
			addr := conn.RemoteAddr().String()
			fmt.Printf("Accepted new client, %v\n", addr)
			allClients[conn] = addr
			go read(conn, messages, deadConnections)

		case conn := <-deadConnections:
			fmt.Printf("Client %v disconnected", allClients[conn])
			delete(allClients, conn)

		case message := <-messages:
			go decodeMsg(message, rxChannels)
		}
	}
}

func acceptConnections(server net.Listener, newConnections chan net.Conn) {
	for {
		conn, err := server.Accept()
		if err != nil {
			fmt.Println(err)
		}
		newConnections <- conn
	}
}
*/

/*
distribute all hall requests
needs to receive ack from each elevator sendt to.
probably need to give it the TCP conn array

func DistributeHallRequests(assignedHallReq) {
	//TODO: all
}


distribute all button lights assosiated with each hallreq at each local elevator
needs to receive ack from each elevator sendt to.
probably need to give it the TCP conn array.
will need ack here aswell as hall req button lights need to be syncronized across computers
func DistributeHallButtonLights(assignedHallReq) {
	//TODO: all
}
*/

func ConnectedToNetwork() bool {
	conn, err := net.Dial("udp", "8.8.8.8:53") // (8.8.8.8 is a Google DNS)
	if err != nil {
		return false
	}
	defer conn.Close()
	return true
}

func RestartOnReconnect(CabCopyCh chan [elevio.N_Floors][elevio.N_Buttons]bool) {
	var CabCopy [elevio.N_Floors]bool
	prevWasConnected := ConnectedToNetwork()

	for {
		select {
		case requestCopy := <-CabCopyCh:
			for floor := 0; floor < elevio.N_Floors; floor++ {
				CabCopy[floor] = requestCopy[floor][elevio.B_Cab]
			}

			fmt.Println("copy cab is: ", elevio.CabArrayToString(CabCopy))
		default:
			if ConnectedToNetwork() && !prevWasConnected {
				cabString := elevio.CabArrayToString(CabCopy)

				command := fmt.Sprintf("gnome-terminal -- go run ./main.go %s", cabString)

				cmd := exec.Command("bash", "-c", command)

				err := cmd.Run()
				if err != nil {
					fmt.Println("Failed to execute command:", err)
				}
				fmt.Println("copy cab is: ", elevio.CabArrayToString(CabCopy))
				panic("No network connection. Terminating current run - restarting from restart.go")
			}
			if ConnectedToNetwork() {
				prevWasConnected = true
			} else {
				prevWasConnected = false
			}
			// Sleep(1 second) was here, should it remain? - 14.03 22:04 - Sveinung og Mikael
		}
		time.Sleep(1 * time.Millisecond)
	}
}

func GetLocalIPv4() string {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		fmt.Println("Error fetching local IPv4 address: ", err)
		return ""
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)

	return localAddr.IP.String()
}

func StringPortToInt(port string) int {
	portWithoutColon := strings.TrimPrefix(port, ":")

	portInt, err := strconv.Atoi(portWithoutColon)
	if err != nil {
		panic(err)
	}
	return portInt
}
