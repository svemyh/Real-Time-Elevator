package primary_backup

import (
	"elevator/config"
	"elevator/conn"
	"elevator/elevator"
	"elevator/elevio"
	"elevator/hall_request_assigner"
	"elevator/network"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"strings"
	"time"
)

type MessageType string

type typeTaggedJSON struct {
	TypeId string
	JSON   []byte
}

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

func UpdateCombinedHallRequests(ActiveElevatorsMap map[string]elevator.Elevator, CombinedHallRequests [elevio.N_Floors][2]bool) [elevio.N_Floors][2]bool {

	for _, elev := range ActiveElevatorsMap {
		for floor := 0; floor < elevio.N_Floors; floor++ {
			CombinedHallRequests[floor][0] = CombinedHallRequests[floor][0] || elev.Requests[floor][0]
			CombinedHallRequests[floor][1] = CombinedHallRequests[floor][1] || elev.Requests[floor][1]
		}
	}
	return CombinedHallRequests
}

func UDPBroadCastPrimaryRole(p string, transmitEnable <-chan bool) {

	port := network.StringPortToInt(p)
	key := "I'm the Primary"

	conn := conn.DialBroadcastUDP(port) // FIX SO THAT ITS COMPATIBLE WITH STRING

	addr, err := net.ResolveUDPAddr("udp4", fmt.Sprintf("255.255.255.255:%d", port))
	if err != nil {
		panic(err)
	}
	enable := true
	for {

		select {
		case enable = <-transmitEnable:
		case <-time.After(200 * time.Millisecond):
		}
		if enable {
			conn.WriteTo([]byte(key), addr)
			//fmt.Println("Printing key in UDP")
		}
	}
}

func UDPBroadCastCombinedHallRequests(port string, CombinedHallRequests [elevio.N_Floors][2]bool, BroadcastCombinedHallRequestsCh chan [elevio.N_Floors][2]bool) {

	conn := conn.DialBroadcastUDP(network.StringPortToInt(port)) // FIX SO THAT ITS COMPATIBLE WITH STRING
	addr, err := net.ResolveUDPAddr("udp4", fmt.Sprintf("255.255.255.255:%d", network.StringPortToInt(port)))
	if err != nil {
		fmt.Println("**Error in UDPBroadCastCombinedHallRequests:", err)
	}
	ticker := time.Tick(200 * time.Millisecond)

	for {
		select { // Blocks until signal recieved on either of these
		case combinedHallRequests := <-BroadcastCombinedHallRequestsCh:
			CombinedHallRequests = combinedHallRequests
		case <-ticker:
			SendCombinedHallRequests(conn, addr, CombinedHallRequests)
		}
	}
}

func AmIPrimary(addressString string, peerUpdateCh chan<- ClientUpdate) (bool, string) {
	port := network.StringPortToInt(addressString)

	conn := conn.DialBroadcastUDP(port)
	conn.SetReadDeadline(time.Now().Add(config.UDPINTERVAL))

	for {
		buffer := make([]byte, config.BUFSIZE)
		n, addr, err := conn.ReadFrom(buffer)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				log.Printf("Timeout reached without receiving %s, becoming primary...", buffer[:n])
				return true, "none"
			}
			log.Printf("Error reading from UDP: %v\n", err)
			return false, "err"
		}

		message := string(buffer[:n])
		if message == "I'm the Primary" {
			log.Printf("Received %s from primary, remaining as client...", message)
			fmt.Println("AmIPrimary determines that primary's formatted address is: ", strings.Split(addr.String(), ":")[0])
			return false, strings.Split(addr.String(), ":")[0]
		}
		// If received message is not "HelloWorld", keep listening until timeout
	}
}

func TCPListenForNewElevators(TCPPort string, clientUpdateCh chan<- ClientUpdate, StateUpdateCh chan hall_request_assigner.ActiveElevator, HallOrderCompleteCh chan elevio.ButtonEvent, DisconnectedElevatorCh chan string, AssignedHallRequestsCh chan map[string][elevio.N_Floors][elevio.N_Buttons - 1]bool, ConsumerChannels map[net.Conn]chan map[string][elevio.N_Floors][elevio.N_Buttons - 1]bool, CloseConnCh chan string, connChan chan net.Conn) {
	ls, err := net.Listen("tcp", TCPPort)
	if err != nil {
		fmt.Println("The connection failed. Error:", err)
		return
	}
	defer ls.Close()
	fmt.Println("Primary is listening for new connections to port:", TCPPort)

	// When AssignedHallRequestsCh recieves a message, StartBroadcaster() distributes it to each of the personalAssignedHallRequestsCh used in TCPWriteElevatorStates()
	go StartBroadcaster(AssignedHallRequestsCh, ConsumerChannels)
	go handleConn(connChan, DisconnectedElevatorCh, CloseConnCh)

	// go handleConn(ConnCloseConsumerChannels)
	for {
		conn, err := ls.Accept()
		if err != nil {
			fmt.Println("Error: ", err)
			continue
		}
		fmt.Println("Conection established to: ", conn.RemoteAddr())
		personalAssignedHallRequestsCh := make(chan map[string][elevio.N_Floors][elevio.N_Buttons - 1]bool, 1024)
		connChan <- conn
		ConsumerChannels[conn] = personalAssignedHallRequestsCh
		go TCPWriteElevatorStates(conn, personalAssignedHallRequestsCh)
		go TCPReadElevatorStates(conn, StateUpdateCh, HallOrderCompleteCh, DisconnectedElevatorCh)

		time.Sleep(100 * time.Millisecond)
	}
}

func handleConn(connChan chan net.Conn, DisconnectedElevatorCh chan string, CloseConnCh chan string) {

	ActiveConns := make(map[string]net.Conn)
	for {
		select {
		case c := <-connChan:
			fmt.Println("RemoteAddr: ", strings.Split((c).RemoteAddr().String(), ":")[0])
			fmt.Println("LocalAddr: ", strings.Split((c).LocalAddr().String(), ":")[0])
			ActiveConns[strings.Split((c).RemoteAddr().String(), ":")[0]] = c

		case c := <-CloseConnCh:
			fmt.Println("c := <-CloseConnCh: ", c)
			for ip, conn := range ActiveConns {
				if ip == c {
					conn.Close()
					delete(ActiveConns, ip)
					DisconnectedElevatorCh <- ip
				}
			}
		}
	}
}

func PrimaryRoutine(ActiveElevatorMap map[string]elevator.Elevator,
	CombinedHallRequests [elevio.N_Floors][elevio.N_Buttons - 1]bool,
	StateUpdateCh chan hall_request_assigner.ActiveElevator,
	HallOrderCompleteCh chan elevio.ButtonEvent,
	DisconnectedElevatorCh chan string,
	AssignHallRequestsCh chan map[string][elevio.N_Floors][elevio.N_Buttons - 1]bool,
	AckCh chan bool,
	ConsumerChannels map[net.Conn]chan map[string][elevio.N_Floors][elevio.N_Buttons - 1]bool,
	connChan chan net.Conn) {

	clientTxEnable := make(chan bool)
	clientUpdateCh := make(chan ClientUpdate)
	helloRx := make(chan ElevatorSystemChannels)
	BroadcastCombinedHallRequestsCh := make(chan [elevio.N_Floors][elevio.N_Buttons - 1]bool, 1024)
	CloseConnCh := make(chan string, 1024)

	go UDPCheckPeerAliveStatus(config.LOCAL_ELEVATOR_ALIVE_PORT, DisconnectedElevatorCh, CloseConnCh)
	go UDPBroadCastPrimaryRole(config.DETECTION_PORT, clientTxEnable)
	go UDPBroadCastCombinedHallRequests(config.HALL_LIGHTS_PORT, CombinedHallRequests, BroadcastCombinedHallRequestsCh)                                                                            //Continously broadcast that you are a primary on UDP
	go TCPListenForNewElevators(config.TCP_LISTEN_PORT, clientUpdateCh, StateUpdateCh, HallOrderCompleteCh, DisconnectedElevatorCh, AssignHallRequestsCh, ConsumerChannels, CloseConnCh, connChan) //Continously listen if new elevator entring networks is trying to establish connection
	go HandlePrimaryTasks(ActiveElevatorMap, CombinedHallRequests, StateUpdateCh, HallOrderCompleteCh, DisconnectedElevatorCh, AssignHallRequestsCh, AckCh, BroadcastCombinedHallRequestsCh)

	for {
		select {
		case c := <-clientUpdateCh:
			fmt.Printf("Client update:\n")
			fmt.Printf("  Clients:    %q\n", c.Client)
			fmt.Printf("  New:      %q\n", c.New)
			fmt.Printf("  Lost:     %q\n", c.Lost)

		case a := <-helloRx:
			fmt.Printf("Received: %#v\n", a)
		}
	}
}

func HandlePrimaryTasks(ActiveElevatorMap map[string]elevator.Elevator,
	CombinedHallRequests [elevio.N_Floors][elevio.N_Buttons - 1]bool,
	StateUpdateCh chan hall_request_assigner.ActiveElevator,
	HallOrderCompleteCh chan elevio.ButtonEvent,
	DisconnectedElevatorCh chan string,
	AssignHallRequestsCh chan map[string][elevio.N_Floors][elevio.N_Buttons - 1]bool,
	AckCh chan bool,
	BroadcastCombinedHallRequestsCh chan [elevio.N_Floors][elevio.N_Buttons - 1]bool) {

	BackupAddr := ""
	var backupConn net.Conn

	if len(ActiveElevatorMap) > 0 { // Guarantees that when Backup is promoted to Primary, that the current active hall requests are redistributed without the necessity of an event occuring (i.e. button pressed, floor arrival, elevator disconnected)
		StateUpdateCh <- hall_request_assigner.ActiveElevator{
			Elevator:  ActiveElevatorMap[GetMapKey(ActiveElevatorMap)],
			MyAddress: GetMapKey(ActiveElevatorMap),
		}
	}
	for {
		// Guarantees that ActiveElevatorMap contains Primary
		if _, exists := ActiveElevatorMap[network.GetLocalIPv4()]; !exists && len(ActiveElevatorMap) > 0 { // TODO: add case for if len=0
			fmt.Println("Primary is not in ActiveElevatorMap. Adding it.")
			StateUpdateCh <- hall_request_assigner.ActiveElevator{
				Elevator:  ActiveElevatorMap[GetMapKey(ActiveElevatorMap)],
				MyAddress: network.GetLocalIPv4(),
			}
		}

		fmt.Println("~~ HandlePrimaryTasks() - ActiveElevatorMap: ", ActiveElevatorMap)
		fmt.Println("~~ HandlePrimaryTasks() - CombinedHallRequests: ", CombinedHallRequests)
		select {
		case stateUpdate := <-StateUpdateCh:
			fmt.Println("StateUpdate: ", stateUpdate)
			ActiveElevatorMap[stateUpdate.MyAddress] = stateUpdate.Elevator

			if len(ActiveElevatorMap) >= 2 {
				if _, exists := ActiveElevatorMap[BackupAddr]; !exists {
					fmt.Println("Backup does not exists yet. Initializing it..")
					BackupAddr = GetBackupAddress(ActiveElevatorMap)
					backupConn = TCPDial(BackupAddr, config.TCP_BACKUP_PORT)
					go TCPReadACK(backupConn, DisconnectedElevatorCh, AckCh) // Using the established backupConn start listening for ACK's from Backup.
					for ip, elev := range ActiveElevatorMap {
						TCPSendActiveElevator(backupConn, hall_request_assigner.ActiveElevator{Elevator: elev, MyAddress: ip})
					}
				}
				TCPSendActiveElevator(backupConn, stateUpdate)
				go func() {
					select {
					case <-AckCh:
						fmt.Println("ACK received: In case stateUpdate")
						CombinedHallRequests = UpdateCombinedHallRequests(ActiveElevatorMap, CombinedHallRequests)
						BroadcastCombinedHallRequestsCh <- CombinedHallRequests
						AssignHallRequestsCh <- hall_request_assigner.HallRequestAssigner(ActiveElevatorMap, CombinedHallRequests)
						fmt.Println("BroadcastCombinedHallRequestsCh <- : ", CombinedHallRequests)
						fmt.Println("AssignHallRequestsCh <- : ", hall_request_assigner.HallRequestAssigner(ActiveElevatorMap, CombinedHallRequests))
					case <-time.After(5 * time.Second):
						fmt.Println("No ACK recieved - Timeout occurred. In case stateUpdate")
						// Handle the timeout event, e.g., retransmit the message or take appropriate action -> i.e. Consider the backup to be dead
					}
				}()
			}

		case completedOrder := <-HallOrderCompleteCh:
			fmt.Println("\n---- Order completed at floor:", completedOrder)

			if len(ActiveElevatorMap) >= 2 {
				if _, exists := ActiveElevatorMap[BackupAddr]; !exists {
					fmt.Println("Backup does not exists yet. Initializing it..")
					BackupAddr = GetBackupAddress(ActiveElevatorMap)
					backupConn = TCPDial(BackupAddr, config.TCP_BACKUP_PORT)
					go TCPReadACK(backupConn, DisconnectedElevatorCh, AckCh) // Using the established backupConn start listening for ACK's from Backup.
					for ip, elev := range ActiveElevatorMap {
						TCPSendActiveElevator(backupConn, hall_request_assigner.ActiveElevator{Elevator: elev, MyAddress: ip})
					}
				}
				TCPSendButtonEvent(backupConn, completedOrder)
				go func() {
					select {
					case <-AckCh:
						fmt.Println("ACK received: In case completedOrder")
						CombinedHallRequests[completedOrder.Floor][completedOrder.Button] = false
						BroadcastCombinedHallRequestsCh <- CombinedHallRequests
					case <-time.After(5 * time.Second):
						fmt.Println("No ACK recieved - Timeout occurred. In case completedOrder")
						// Handle the timeout event, e.g., retransmit the message or take appropriate action -> i.e. Consider the backup to be dead
					}
				}()
			}

		case disconnectedElevator := <-DisconnectedElevatorCh:
			fmt.Println("In case disconnectedElevator: recieved:", disconnectedElevator)
			fmt.Println("In case disconnectedElevator:formatted:", strings.Split(disconnectedElevator, ":")[0])
			delete(ActiveElevatorMap, strings.Split(disconnectedElevator, ":")[0])
			fmt.Println("after removing elevator, active elev map is: ", ActiveElevatorMap)
			fmt.Println("1) ", strings.Split(disconnectedElevator, ":")[0])
			fmt.Println("2) ", disconnectedElevator)
			fmt.Println("3) ", BackupAddr)

			if strings.Split(disconnectedElevator, ":")[0] == BackupAddr {
				fmt.Println("Backup disconnected. Removing it ")
				BackupAddr = ""
				backupConn.Close()
				fmt.Println("4) ", BackupAddr)
			}

			if len(ActiveElevatorMap) >= 2 {
				if _, exists := ActiveElevatorMap[BackupAddr]; !exists {
					fmt.Println("Backup does not exists yet. Initializing it..")
					BackupAddr = GetBackupAddress(ActiveElevatorMap)
					backupConn = TCPDial(BackupAddr, config.TCP_BACKUP_PORT)
					go TCPReadACK(backupConn, DisconnectedElevatorCh, AckCh) // Using the established backupConn start listening for ACK's from Backup.
					for ip, elev := range ActiveElevatorMap {
						TCPSendActiveElevator(backupConn, hall_request_assigner.ActiveElevator{Elevator: elev, MyAddress: ip})
					}
				}

				TCPSendString(backupConn, disconnectedElevator) // Gives notice to backup of disconnected elevator
				go func() {
					select {
					case <-AckCh:
						fmt.Println("ACK received: In case stateUpdate")
						AssignHallRequestsCh <- hall_request_assigner.HallRequestAssigner(ActiveElevatorMap, CombinedHallRequests)
					case <-time.After(5 * time.Second):
						fmt.Println("No ACK recieved - Timeout occurred. In case stateUpdate")
						// Handle the timeout event, e.g., retransmit the message or take appropriate action -> i.e. Consider the backup to be dead
					}
				}()
			}
		}
	}
}

func GetBackupAddress(ActiveElevatorMap map[string]elevator.Elevator) string {
	for key, _ := range ActiveElevatorMap {
		if key != network.GetLocalIPv4() {
			return key
		}
	}
	return "err"
}

func GetMapKey(ActiveElevatorMap map[string]elevator.Elevator) string {
	for key := range ActiveElevatorMap {
		return key
	}
	return ""
}

func TCPSendActiveElevator(conn net.Conn, activeElevator hall_request_assigner.ActiveElevator) error {
	myActiveElevatorMsg := MsgActiveElevator{
		Type:    TypeActiveElevator,
		Content: activeElevator,
	}
	fmt.Println("my_ActiveElevatorMsg:", myActiveElevatorMsg)
	data, err := json.Marshal(myActiveElevatorMsg)

	if err != nil {
		fmt.Println("Error encoding ActiveElevator to json: ", err)
		return err
	}

	_, err = conn.Write(data)
	if err != nil {
		fmt.Println("Error sending ActiveElevator: ", err)
		return err
	}
	time.Sleep(50 * time.Millisecond)
	return nil
}

func TCPSendButtonEvent(conn net.Conn, buttonEvent elevio.ButtonEvent) error {
	myButtonEventMsg := MsgButtonEvent{
		Type:    TypeButtonEvent,
		Content: buttonEvent,
	}

	data, err := json.Marshal(myButtonEventMsg)
	if err != nil {
		fmt.Println("Error encoding ButtonEvent to json: ", err)
		return err
	}

	_, err = conn.Write(data)
	if err != nil {
		fmt.Println("Error sending ButtonEvent: ", err)
		return err
	}
	time.Sleep(50 * time.Millisecond)
	return nil
}

func TCPSendACK(conn net.Conn) {
	myACKMsg := MsgACK{
		Type:    TypeACK,
		Content: true,
	}

	fmt.Println("TCPSendACK():", myACKMsg)
	data, err := json.Marshal(myACKMsg)
	if err != nil {
		fmt.Println("Error encoding ACK to json: ", err)
		return
	}

	_, err = conn.Write(data)
	if err != nil {
		fmt.Println("Error sending ACK: ", err)
		return
	}
	time.Sleep(50 * time.Millisecond)
}

func TCPSendString(conn net.Conn, str string) {
	myStringMsg := MsgString{
		Type:    TypeString,
		Content: str,
	}

	fmt.Println("TCPSendString():", myStringMsg)
	data, err := json.Marshal(myStringMsg)
	if err != nil {
		fmt.Println("Error encoding MsgString to json: ", err)
		return
	}

	_, err = conn.Write(data)
	if err != nil {
		fmt.Println("Error sending MsgString: ", err)
		return
	}
	time.Sleep(50 * time.Millisecond)
}

func SendCombinedHallRequests(conn net.PacketConn, UDPaddr *net.UDPAddr, CombinedHallRequests [elevio.N_Floors][elevio.N_Buttons - 1]bool) {
	myMsgCombinedHallRequests := MsgCombinedHallRequests{
		Type:    TypeCombinedHallRequests,
		Content: CombinedHallRequests,
	}

	data, err := json.Marshal(myMsgCombinedHallRequests) // TODO: REFACTOR - Do definition of MsgCombinedHallRequests inline here.
	if err != nil {
		fmt.Println("Error encoding MsgCombinedHallRequests to json: ", err)
		return
	}

	_, err = conn.WriteTo(data, UDPaddr)
	if err != nil {
		fmt.Println("Error sending MsgCombinedHallRequests: ", err)
		return
	}

	time.Sleep(50 * time.Millisecond)
}

func UDPReadCombinedHallRequests(port string) {
	conn := conn.DialBroadcastUDP(network.StringPortToInt(port))

	defer conn.Close()
	for {
		var buf [config.BUFSIZE]byte
		n, _, err := conn.ReadFrom(buf[:])
		if err != nil {
			fmt.Println("Error reading from connection: ", err)
			continue
		}

		var genericMsg map[string]interface{}
		if err := json.Unmarshal(buf[:n], &genericMsg); err != nil {
			fmt.Println("Error unmarshaling generic message: ", err)
			fmt.Println("Panic was here 2)")
			continue
		}

		switch MessageType(genericMsg["type"].(string)) {
		case TypeCombinedHallRequests:
			var msg MsgCombinedHallRequests
			if err := json.Unmarshal(buf[:n], &msg); err != nil {
				fmt.Println("UDPReadCombinedHallRequests() - Error unmarshaling MsgCombinedHallRequests from Primary: ", err)
				continue
			}
			SetAllHallLights(msg.Content)

		default:
			fmt.Println("Unknown message type recieved")
		}
		time.Sleep(50 * time.Millisecond)
	}
}

func SetAllHallLights(CombinedHallRequests [elevio.N_Floors][elevio.N_Buttons - 1]bool) {
	for floor := 0; floor < elevio.N_Floors; floor++ {
		for btn := 0; btn < 2; btn++ {
			elevio.SetButtonLamp(elevio.ButtonType(btn), floor, CombinedHallRequests[floor][btn])
		}
	}
}

// Alias: Server()
func TCPReadACK(conn net.Conn, DisconnectedElevatorCh chan string, AckCh chan bool) {
	//TODO:Read the states and store in a buffer
	//TODO: Check if the read data was due to local elevator reaching a floor and clearing a request (send cleared request on OrderCompleteCh)
	//TODO:send the updated states on stateUpdateCh so that it can be read in HandlePrimaryTasks(StateUpdateCh)
	// Can be added/expanded: LocalErrorDetectedCh or similar
	// type StateUpdateCh = IP + elevatorStates
	// type HallOrderCopleteCh = floor number (of cab call completed)

	fmt.Printf("TCPReadACK() - *New connection accepted from address: %s\n", conn.LocalAddr())
	fmt.Printf("TCPReadACK() - *New connection accepted from address: %s to %s\n", conn.LocalAddr(), conn.RemoteAddr().String())

	defer conn.Close()

	for {
		// Create buffer and read data into the buffer using conn.Read()
		var buf [config.BUFSIZE]byte
		n, err := conn.Read(buf[:])
		if err != nil {
			// Error means TCP-conn has broken -> Need to feed this signal to drop the conn's respective ActiveElevator from Primary's ActiveElevators. It is now considered inactive.
			DisconnectedElevatorCh <- conn.RemoteAddr().String()
			break
		}

		// Decoding said data into a json-style object
		var genericMsg map[string]interface{}
		if err := json.Unmarshal(buf[:n], &genericMsg); err != nil {
			fmt.Println("Error unmarshaling generic message: ", err)
			fmt.Println("Panic was here 3)")
			continue
		}
		// Based on MessageType (which is an element of each struct sent over connection) determine how its corresponding data should be decoded.
		switch MessageType(genericMsg["type"].(string)) {
		case TypeACK:
			var msg MsgACK
			if err := json.Unmarshal(buf[:n], &msg); err != nil {
				panic(err)
			}
			AckCh <- msg.Content

		default:
			fmt.Println("Unknown message type")
		}
		time.Sleep(50 * time.Millisecond)
	}
}

// ____________________________________________________________________________________________________________________________________________

// Alias: RunPrimaryBackup()
func InitNetwork(FSMStateUpdateCh chan hall_request_assigner.ActiveElevator, FSMHallOrderCompleteCh chan elevio.ButtonEvent, StateUpdateCh chan hall_request_assigner.ActiveElevator, HallOrderCompleteCh chan elevio.ButtonEvent, DisconnectedElevatorCh chan string, FSMAssignedHallRequestsCh chan [elevio.N_Floors][elevio.N_Buttons - 1]bool, AssignHallRequestsCh chan map[string][elevio.N_Floors][elevio.N_Buttons - 1]bool, AckCh chan bool) {
	clientUpdateCh := make(chan ClientUpdate)
	//clientTxEnable := make(chan bool)
	var id string

	isPrimary, primaryAddress := AmIPrimary(config.DETECTION_PORT, clientUpdateCh)
	if isPrimary {
		if id == "" {
			localIP := network.GetLocalIPv4()
			id = fmt.Sprintf("Primary-%s-%d", localIP, os.Getpid())
			fmt.Printf("My id: %s\n", id)
		}
		log.Println("Operating as Primary...")

		var CombinedHallRequests [elevio.N_Floors][elevio.N_Buttons - 1]bool
		ActiveElevatorMap := make(map[string]elevator.Elevator)
		ConsumerChannels := make(map[net.Conn]chan map[string][elevio.N_Floors][elevio.N_Buttons - 1]bool, 1024)
		ConnChan := make(chan net.Conn, 1024)
		go PrimaryRoutine(ActiveElevatorMap, CombinedHallRequests, StateUpdateCh, HallOrderCompleteCh, DisconnectedElevatorCh, AssignHallRequestsCh, AckCh, ConsumerChannels, ConnChan)
		time.Sleep(1500 * time.Millisecond)
		TCPDialPrimary(network.GetLocalIPv4()+config.TCP_LISTEN_PORT, FSMStateUpdateCh, FSMHallOrderCompleteCh, FSMAssignedHallRequestsCh)
	} else {
		if id == "" {
			localIP := network.GetLocalIPv4()
			id = fmt.Sprintf("Client-%s-%d\n", localIP, os.Getpid())
			fmt.Printf("My id: %s", id)
		}
		log.Println("Operating as client...")
		go TCPDialPrimary(primaryAddress+config.TCP_LISTEN_PORT, FSMStateUpdateCh, FSMHallOrderCompleteCh, FSMAssignedHallRequestsCh)
		go TCPListenForNewPrimary(config.TCP_NEW_PRIMARY_LISTEN_PORT, FSMStateUpdateCh, FSMHallOrderCompleteCh, FSMAssignedHallRequestsCh)
		conn, err := TCPListenForBackupPromotion(config.TCP_BACKUP_PORT) //will simply be a net.Listen("TCP", "primaryAdder"). This blocks code until a connection is established
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
		var buf [config.BUFSIZE]byte
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
		var buf [config.BUFSIZE]byte
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
			fmt.Println("Panic was here 1)")
			continue
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

func TCPDial(address string, port string) net.Conn {
	conn, err := net.Dial("tcp", address+port)
	if err != nil {
		fmt.Println("Connection failed. Error: ", err)
		return nil
	}
	return conn
}

func UDPCheckPeerAliveStatus(port string, DisconnectedElevatorCh chan string, CloseConnCh chan string) {
	conn := conn.DialBroadcastUDP(network.StringPortToInt(port))
	checkAliveStatus := make(map[string]int)

	defer conn.Close()
	for {
		var buf [config.BUFSIZE]byte
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
				CloseConnCh <- IP
			}
		}

		time.Sleep(50 * time.Millisecond)
	}
}

func RestartOnReconnect(CabCopyCh chan [elevio.N_Floors][elevio.N_Buttons]bool) {
	var CabCopy [elevio.N_Floors]bool
	prevWasConnected := network.ConnectedToNetwork()

	for {
		select {
		case requestCopy := <-CabCopyCh:
			for floor := 0; floor < elevio.N_Floors; floor++ {
				CabCopy[floor] = requestCopy[floor][elevio.B_Cab]
			}

			fmt.Println("copy cab is: ", elevio.CabArrayToString(CabCopy))
		default:
			if network.ConnectedToNetwork() && !prevWasConnected {
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
			if network.ConnectedToNetwork() {
				prevWasConnected = true
			} else {
				prevWasConnected = false
			}
			// Sleep(1 second) was here, should it remain? - 14.03 22:04 - Sveinung og Mikael
		}
		time.Sleep(1 * time.Millisecond)
	}
}
