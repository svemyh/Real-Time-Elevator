package network

import (
	"elevator/conn"
	"elevator/elevator"
	"elevator/elevio"
	"elevator/hall_request_assigner"
	"encoding/json"
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
var LOCAL_ELEVATOR_ALIVE_PORT string = ":17878"

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

type MsgACK struct {
	Type    MessageType `json:"type"`
	Content bool        "json:content"
}

type MsgString struct {
	Type    MessageType `json:"type"`
	Content string      "json:content"
}

type MsgCombinedHallRequests struct {
	Type    MessageType                                 `json:"type"`
	Content [elevio.N_Floors][elevio.N_Buttons - 1]bool "json:content"
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
		ConsumerChannels := make(map[net.Conn]chan map[string][elevio.N_Floors][elevio.N_Buttons - 1]bool, 1024)
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
		BackupRoutine(conn, StateUpdateCh, HallOrderCompleteCh, DisconnectedElevatorCh, AssignHallRequestsCh, AckCh)
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

		go RecieveAssignedHallRequests(conn, FSMAssignedHallRequestsCh)
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

	go RecieveAssignedHallRequests(conn, FSMAssignedHallRequestsCh)
	sendLocalStatesToPrimaryLoop(conn, FSMStateUpdateCh, FSMHallOrderCompleteCh)
}

func RecieveAssignedHallRequests(conn net.Conn, FSMAssignedHallRequestsCh chan [elevio.N_Floors][elevio.N_Buttons - 1]bool) { // NOT TESTED!
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
			err := TCPSendActiveElevator(conn, stateUpdate)
			if err != nil {
				return
			}

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

func TCPWriteElevatorStates(conn net.Conn, personalAssignedHallRequestsCh chan map[string][elevio.N_Floors][elevio.N_Buttons - 1]bool) {
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

// Recieves message on AssignedHallRequestsCh and distributes said message to all consumer go-routines ConsumerAssignedHallRequestsCh
func StartBroadcaster(AssignedHallRequestsCh chan map[string][elevio.N_Floors][elevio.N_Buttons - 1]bool, Consumers map[net.Conn]chan map[string][elevio.N_Floors][elevio.N_Buttons - 1]bool) {
	for hallRequests := range AssignedHallRequestsCh {
		for _, ch := range Consumers {
			ch <- hallRequests
		}
	}
}

// Alias: Server()
func TCPReadElevatorStates(conn net.Conn, StateUpdateCh chan hall_request_assigner.ActiveElevator, HallOrderCompleteCh chan elevio.ButtonEvent, DisconnectedElevatorCh chan string) {
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

func UDPCheckPeerAliveStatus(port string, DisconnectedElevatorCh chan string, CloseConnCh chan string) {
	conn := conn.DialBroadcastUDP(StringPortToInt(port))
	checkAliveStatus := make(map[string]int)

	defer conn.Close()
	for {
		var buf [bufSize]byte
		n, _, err := conn.ReadFrom(buf[:])
		if err != nil {
			fmt.Println("Error reading from UPDCheckAliveStatus: ", err)
			continue
		}
		peerIP := string(buf[:n])

		if _, exists := checkAliveStatus[peerIP]; !exists {
			checkAliveStatus[peerIP] = 0
		}

		log.Print("alive map: ", checkAliveStatus)

		for IP, _ := range checkAliveStatus {
			if IP != peerIP {
				checkAliveStatus[IP]++
			} else {
				checkAliveStatus[IP] = 0
			}
		}

		for IP, count := range checkAliveStatus {
			if count > 30 {
				//send IP on disconnected elevators channel
				print("detected a disconnected elevator with IP: ", IP)
				delete(checkAliveStatus, IP)
				CloseConnCh <- peerIP
			}
		}

		time.Sleep(50 * time.Millisecond)
	}
}

func UDPBroadcastAlive(p string) {

	port := StringPortToInt(p)
	key := GetLocalIPv4()

	conn := conn.DialBroadcastUDP(port) // FIX SO THAT ITS COMPATIBLE WITH STRING

	addr, err := net.ResolveUDPAddr("udp4", fmt.Sprintf("255.255.255.255:%d", port))
	if err != nil {
		panic(err)
	}

	for {
		conn.WriteTo([]byte(key), addr)
		fmt.Println("Broadcasting status!")
		time.Sleep(100 * time.Millisecond)
	}
}

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
