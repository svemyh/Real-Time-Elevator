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
	"strings"
	"time"
)

type typeTaggedJSON struct { // UNUSED?
	TypeId string
	JSON   []byte
}

/*
Decides if a computer running on network should become primary.
Listens for a broadcast from primary for a RANDOMIZED SMALL AMOUNT OF TIME
(if not two elevators can decide to become primary at same time if started at same time)
If nothing is heard in this time ---> returns True
Return:
bool
*/

// OR's the new hall requests

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

	port := StringPortToInt(p)
	key := "OptimusPrime"

	conn := conn.DialBroadcastUDP(port) // FIX SO THAT ITS COMPATIBLE WITH STRING

	addr, _ := net.ResolveUDPAddr("udp4", fmt.Sprintf("255.255.255.255:%d", port))
	enable := true
	for {

		select {
		case enable = <-transmitEnable:
		case <-time.After(udpInterval / 4):
		}
		if enable {
			conn.WriteTo([]byte(key), addr)
			//fmt.Println("Printing key in UDP")
		}
	}
}

func UDPBroadCastCombinedHallRequests(port string, CombinedHallRequests [elevio.N_Floors][elevio.N_Buttons - 1]bool, BroadcastCombinedHallRequestsCh chan [elevio.N_Floors][elevio.N_Buttons - 1]bool) {

	conn := conn.DialBroadcastUDP(StringPortToInt(port)) // FIX SO THAT ITS COMPATIBLE WITH STRING
	addr, err := net.ResolveUDPAddr("udp4", fmt.Sprintf("255.255.255.255:%d", StringPortToInt(port)))
	if err != nil {
		fmt.Println("**Error in UDPBroadCastCombinedHallRequests:", err)
	}
	ticker := time.Tick(100 * time.Millisecond)

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
	port := StringPortToInt(addressString)

	conn := conn.DialBroadcastUDP(port)
	conn.SetReadDeadline(time.Now().Add(udpInterval))

	for {
		buffer := make([]byte, bufSize)
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
		if message == "OptimusPrime" {
			log.Printf("Received %s from primary, remaining as client...", message)
			fmt.Println("AmIPrimary determines that primary's formatted address is: ", strings.Split(addr.String(), ":")[0])
			return false, strings.Split(addr.String(), ":")[0]
		}
		// If received message is not "HelloWorld", keep listening until timeout
	}
}

func TCPListenForNewElevators(TCPPort string, E ElevatorSystemChannels) {
	//listen for new elevators on TCP port
	//when connection established run the go routine TCPReadElevatorStates to start reading data from the conn
	//go TCPReadElevatorStates(stateUpdateCh)

	ls, err := net.Listen("tcp", TCPPort)
	if err != nil {
		fmt.Println("The connection failed. Error:", err)
		return
	}
	defer ls.Close()

	fmt.Println("Primary is listening for new connections to port:", TCPPort)
	for {
		conn, err := ls.Accept()
		if err != nil {
			fmt.Println("Error: ", err)
			continue
		}

		go TCPReadElevatorStates(conn, E)
		go TCPWriteElevatorStates(conn, E)
		time.Sleep(1 * time.Second)
	}
}

func PrimaryRoutine(E ElevatorSystemChannels) { // Arguments: StateUpdateCh, OrderCompleteCh, ActiveElevators
	//start by establishing TCP connection with yourself (can be done in TCPListenForNewElevators)
	//OR, establish self connection once in RUNPRIMARYBACKUP() and handle selfconnect for future primary in backup.BecomePrimary()

	clientTxEnable := make(chan bool)
	BroadcastCombinedHallRequestsCh := make(chan [elevio.N_Floors][elevio.N_Buttons - 1]bool, bufSize)

	go UDPBroadCastPrimaryRole(DETECTION_PORT, clientTxEnable)
	go UDPBroadCastCombinedHallRequests(HALL_LIGHTS_PORT, E.CombinedHallRequests, BroadcastCombinedHallRequestsCh)                                   //Continously broadcast that you are a primary on UDP
	go TCPListenForNewElevators(TCP_LISTEN_PORT, E) //Continously listen if new elevator entring networks is trying to establish connection
	go HandlePrimaryTasks(E, BroadcastCombinedHallRequestsCh)
}

// get new states everytime a local elevator updates their states.
// We then can operate as a primary alone on network or a primary with other elevators on network
// we start by initializing a backup if possible
// then if we have other elevators on network then assign hall req for each elevator(by cost) distribute them and button lights
// if there are other elevators on network then send states to the backup

func HandlePrimaryTasks(E ElevatorSystemChannels, BroadcastCombinedHallRequestsCh chan [elevio.N_Floors][elevio.N_Buttons - 1]bool) {

	BackupAddr := ""
	var backupConn net.Conn
	//var ActiveElevators []ActiveElevator // init here or take in as param to func, allows Backup.BecomePrimary to send in prev states
	//can send in as empty array first time primary takes over

	if len(E.ActiveElevatorMap) > 0 { // Guarantees that when backup is promoted to primary, that the current active hall requests are redistributed without the necessity of an event occuring (i.e. button pressed, floor arrival, elevator disconnected)
		E.StateUpdateCh <- hall_request_assigner.ActiveElevator{
			Elevator:  E.ActiveElevatorMap[GetMapKey(E.ActiveElevatorMap)],
			MyAddress: GetMapKey(E.ActiveElevatorMap),
		}
	}
	for {
		fmt.Println("~~ HandlePrimaryTasks() - ActiveElevatorMap: ", E.ActiveElevatorMap)
		fmt.Println("~~ HandlePrimaryTasks() - CombinedHallRequests: ", E.CombinedHallRequests)
		select {
		case stateUpdate := <-E.StateUpdateCh: //updates if new state is sendt on one of TCP conns, blocks if not
			//TODO: compare the state update from single elevator to active elevator array and update activeElevators
			//TODO: update some sort of global HALLREQ array with the new hall requests
			fmt.Println("StateUpdate: ", stateUpdate)

			// 0) If stateUpdate.MyAddress not in ActiveElevators -> stateUpdate ActiveElevators.Append()
			// 1) Overwrite the rest of the states (including cab requests) / 0 & 1 effectively same
			E.ActiveElevatorMap[stateUpdate.MyAddress] = stateUpdate.Elevator
			//case len(ActiveElevatorMap) == 1:

			// 2) OR new button-presses in new ActiveElevators to CombinedHallRequests // TODO:
			// hall_request_assigner.ActiveElevators_to_CombinedHallRequests() or similar

			// Pick one of the elevators in ActiveElevatorMap

			// 3) IF not backup -> init a backup; IF backup exists -> send stateInfo to backup
			if len(E.ActiveElevatorMap) >= 2 {
				if _, exists := E.ActiveElevatorMap[BackupAddr]; !exists {
					// init a backup here:
					fmt.Println("Backup does not exists yet. Initializing it..")
					BackupAddr = GetBackupAddress(E.ActiveElevatorMap)
					backupConn = TCPDialBackup(BackupAddr, TCP_BACKUP_PORT)
					go TCPReadACK(backupConn, E) // Using the established backupConn start listening for ACK's from Backup.
				}
				TCPSendActiveElevator(backupConn, stateUpdate) // TODO: Needs to be updated to TCPSendActiveElevatorWithAck() which blocks until ack recieved.
				//This function is only for the backup/primary-communication.
				E.CombinedHallRequests = UpdateCombinedHallRequests(E.ActiveElevatorMap, E.CombinedHallRequests)
				BroadcastCombinedHallRequestsCh <- E.CombinedHallRequests
				go func() {
					select { // Blocks until signal received on either of these
					case <-E.AckCh:
						fmt.Println("ACK received: In case stateUpdate")
						E.AssignHallRequestsMapCh <- hall_request_assigner.HallRequestAssigner(E.ActiveElevatorMap, E.CombinedHallRequests)
					case <-time.After(5 * time.Second):
						fmt.Println("No ACK recieved - Timeout occurred. In case stateUpdate")
						// Handle the timeout event, e.g., retransmit the message or take appropriate action -> i.e. Consider the backup to be dead
					}
				}()
			}

			// For test purposes
			E.CombinedHallRequests = UpdateCombinedHallRequests(E.ActiveElevatorMap, E.CombinedHallRequests)
			BroadcastCombinedHallRequestsCh <- E.CombinedHallRequests
			E.AssignHallRequestsMapCh <- hall_request_assigner.HallRequestAssigner(E.ActiveElevatorMap, E.CombinedHallRequests)

			//if len(ActiveElevators) > 1 {
			//TODO: assign  new backup if needed based based on state update.
			//TODO: send updated states to backup (with ack) (if there still is a backup)
			//TODO: assign new new hall orders for each elevator through cost-func

			//DistributeHallRequests(assignedHallReq)     //Distribute new hall requests to each elevator, needs ack and blocking until done
			//DistributeHallButtonLights(assignedHallReq) //Distribute the button lights to each now that we have ack from each
			//} else {
			//TODO: assign new new hall orders for each elevator through cost-func. we are now a primary alone on network
			//TODO: Should have some check to see if are a primary that lost network (so a new primary has been made) or if we have network connection and no other elevators on net

			//backup = nil
			//DistributeHallRequests()     //Distribute new hall requests to each elevator, needs ack and blocking until done
			//DistributeHallButtonLights() //Distribute the button lights to each now that we have ack from each
			//}

		case completedOrder := <-E.HallOrderCompleteCh:
			//TODO: clear order from some sort of global HALLREQ array
			fmt.Println("\n---- Order completed at floor:", completedOrder)
			E.CombinedHallRequests[completedOrder.Floor][completedOrder.Button] = false
			BroadcastCombinedHallRequestsCh <- E.CombinedHallRequests

			if len(E.ActiveElevatorMap) >= 2 {
				if _, exists := E.ActiveElevatorMap[BackupAddr]; !exists {
					// init a backup here:
					fmt.Println("Backup does not exists yet. Initializing it..")
					BackupAddr = GetBackupAddress(E.ActiveElevatorMap)
					backupConn = TCPDialBackup(BackupAddr, TCP_BACKUP_PORT)
					go TCPReadACK(backupConn, E) // Using the established backupConn start listening for ACK's from Backup.
				}
				TCPSendButtonEvent(backupConn, completedOrder) // Writing to Backup
				// TODO: Wait for ACK
				go func() {
					select { // Blocks until signal recieved on either of these
					case <-E.AckCh:
						// Do nothing
						fmt.Println("ACK received: In case completedOrder")
					case <-time.After(5 * time.Second):
						fmt.Println("No ACK recieved - Timeout occurred. In case completedOrder")
						// Handle the timeout event, e.g., retransmit the message or take appropriate action -> i.e. Consider the backup to be dead
					}
				}()
			}

			//CombinedHallRequests = UpdateCombinedHallRequests(ActiveElevatorMap, CombinedHallRequests)
			//ssignHallRequestsCh <- hall_request_assigner.HallRequestAssigner(ActiveElevatorMap, CombinedHallRequests)

		case disconnectedElevator := <-E.DisconnectedElevatorCh:
			delete(E.ActiveElevatorMap, disconnectedElevator)

			// TODO: Implement TCPSendDisconnectedElevator(disconnectedElevator) // Backup also needs this information
			// WaitForAcknowledgment(AckCh)

			// TODO: if disconnectedElevator == backupElevator DO: initialize a new Backup
			if len(E.ActiveElevatorMap) >= 2 {
				if _, exists := E.ActiveElevatorMap[BackupAddr]; !exists {
					// init a backup here:
					fmt.Println("Backup does not exists yet. Initializing it..")
					BackupAddr = GetBackupAddress(E.ActiveElevatorMap)
					backupConn = TCPDialBackup(BackupAddr, TCP_BACKUP_PORT)
					go TCPReadACK(backupConn, E) // Using the established backupConn start listening for ACK's from Backup.
				}

				TCPSendString(backupConn, disconnectedElevator)
				go func() { // wait for ACK
					select { // Blocks until signal received on either of these
					case <-E.AckCh:
						fmt.Println("ACK received: In case stateUpdate")
						E.AssignHallRequestsMapCh <- hall_request_assigner.HallRequestAssigner(E.ActiveElevatorMap, E.CombinedHallRequests)
					case <-time.After(5 * time.Second):
						fmt.Println("No ACK recieved - Timeout occurred. In case stateUpdate")
						// Handle the timeout event, e.g., retransmit the message or take appropriate action -> i.e. Consider the backup to be dead
					}
				}()
			}
		}
	}
}

func WaitForAcknowledgment(ackCh chan bool) {
	select {
	case <-ackCh:
		return
	case <-time.After(5 * time.Second):
		fmt.Println("Timeout occurred. No ACK received.")
		// Handle the timeout event, e.g., retransmit the message or take appropriate action -> i.e. Consider the backup to be dead
	}
}

func GetBackupAddress(ActiveElevatorMap map[string]elevator.Elevator) string {
	for key := range ActiveElevatorMap {
		if key != GetLocalIPv4() {
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

func TCPSendActiveElevator(conn net.Conn, activeElevator hall_request_assigner.ActiveElevator) {
	myActiveElevatorMsg := MsgActiveElevator{
		Type:    TypeActiveElevator,
		Content: activeElevator,
	}
	fmt.Println("my_ActiveElevatorMsg:", myActiveElevatorMsg)
	data, err := json.Marshal(myActiveElevatorMsg)

	if err != nil {
		fmt.Println("Error encoding ActiveElevator to json: ", err)
		return
	}

	_, err = conn.Write(data)
	if err != nil {
		fmt.Println("Error sending ActiveElevator: ", err)
		return
	}
	time.Sleep(50 * time.Millisecond)
}

func TCPSendButtonEvent(conn net.Conn, buttonEvent elevio.ButtonEvent) {
	myButtonEventMsg := MsgButtonEvent{
		Type:    TypeButtonEvent,
		Content: buttonEvent,
	}

	data, err := json.Marshal(myButtonEventMsg)
	if err != nil {
		fmt.Println("Error encoding ButtonEvent to json: ", err)
		return
	}

	_, err = conn.Write(data)
	if err != nil {
		fmt.Println("Error sending ButtonEvent: ", err)
		return
	}
	time.Sleep(50 * time.Millisecond)
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
	conn := conn.DialBroadcastUDP(StringPortToInt(port))

	defer conn.Close()
	for {
		var buf [bufSize]byte
		n, _, err := conn.ReadFrom(buf[:])
		if err != nil {
			continue
		}

		var genericMsg map[string]interface{}
		if err := json.Unmarshal(buf[:n], &genericMsg); err != nil {
			fmt.Println("Error unmarshaling generic message: ", err)
			log.Fatal(err)
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
func TCPReadACK(conn net.Conn, E ElevatorSystemChannels) {
	//TODO:Read the states and store in a buffer
	//TODO: Check if the read data was due to local elevator reaching a floor and clearing a request (send cleared request on OrderCompleteCh)
	//TODO:send the updated states on stateUpdateCh so that it can be read in HandlePrimaryTasks(StateUpdateCh)
	// Can be added/expanded: LocalErrorDetectedCh or similar
	// type StateUpdateCh = IP + elevatorStates
	// type HallOrderCopleteCh = floor number (of cab call completed)

	fmt.Printf("TCPReadElevatorStates() - *New connection accepted from address: %s\n", conn.LocalAddr())

	defer conn.Close()

	for {
		// Create buffer and read data into the buffer using conn.Read()
		var buf [bufSize]byte
		n, err := conn.Read(buf[:])
		if err != nil {
			// Error means TCP-conn has broken -> Need to feed this signal to drop the conn's respective ActiveElevator from Primary's ActiveElevators. It is now considered inactive.
			E.DisconnectedElevatorCh <- conn.LocalAddr().String()
			log.Fatal(err)
		}

		// Decoding said data into a json-style object
		var genericMsg map[string]interface{}
		if err := json.Unmarshal(buf[:n], &genericMsg); err != nil {
			fmt.Println("Error unmarshaling generic message: ", err)
			log.Fatal(err)
		}
		// Based on MessageType (which is an element of each struct sent over connection) determine how its corresponding data should be decoded.
		switch MessageType(genericMsg["type"].(string)) {
		case TypeACK:
			var msg MsgACK
			if err := json.Unmarshal(buf[:n], &msg); err != nil {
				log.Fatal(err)
			}
			E.AckCh <- msg.Content

		default:
			fmt.Println("Unknown message type")
		}
	}
}
