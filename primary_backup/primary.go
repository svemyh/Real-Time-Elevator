package primary_backup

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
	key := "I'm the Primary"

	conn := conn.DialBroadcastUDP(port)

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
		}
	}
}

func UDPBroadCastCombinedHallRequests(port string, 
	CombinedHallRequests [elevio.N_Floors][2]bool, 
	BroadcastCombinedHallRequestsCh chan [elevio.N_Floors][2]bool,
) {

	conn := conn.DialBroadcastUDP(StringPortToInt(port)) 
	addr, err := net.ResolveUDPAddr("udp4", fmt.Sprintf("255.255.255.255:%d", StringPortToInt(port)))
	if err != nil {
		fmt.Println("Error resolving address:", err)
	}
	ticker := time.Tick(200 * time.Millisecond)

	for {
		select {
		case combinedHallRequests := <-BroadcastCombinedHallRequestsCh:
			CombinedHallRequests = combinedHallRequests
		case <-ticker:
			SendCombinedHallRequests(conn, addr, CombinedHallRequests)
		}
	}
}

func AmIPrimary(addressString string) (bool, string) {
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
		if message == "I'm the Primary" {
			log.Printf("Received %s from primary, remaining as client...", message)
			return false, strings.Split(addr.String(), ":")[0]
		}
	}
}

func TCPListenForNewElevators(TCPPort 					string, 
							  StateUpdateCh 			chan hall_request_assigner.ActiveElevator, 
							  HallOrderCompleteCh 		chan elevio.ButtonEvent, 
							  DisconnectedElevatorCh 	chan string, 
							  AssignedHallRequestsCh 	chan map[string][elevio.N_Floors][elevio.N_Buttons - 1]bool, 
							  ConsumerChannels 			map[net.Conn]chan map[string][elevio.N_Floors][elevio.N_Buttons - 1]bool, 
							  CloseConnCh 				chan string, 
							  connChan 					chan net.Conn,
) {
	ls, err := net.Listen("tcp", TCPPort)
	if err != nil {
		fmt.Println("The connection failed. Error:", err)
		return
	}
	defer ls.Close()
	fmt.Println("Primary is listening for new connections to port:", TCPPort)

	go StartBroadcaster(AssignedHallRequestsCh, ConsumerChannels)
	go handleConn(connChan, DisconnectedElevatorCh, CloseConnCh)

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

func handleConn(connChan 				chan net.Conn, 
				DisconnectedElevatorCh 	chan string, 
				CloseConnCh 			chan string,
) {

	ActiveConns := make(map[string]net.Conn)
	for {
		select {
		case c := <-connChan:
			ActiveConns[strings.Split((c).RemoteAddr().String(), ":")[0]] = c

		case c := <-CloseConnCh:
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

func PrimaryRoutine(ActiveElevatorMap 		map[string]elevator.Elevator,
					CombinedHallRequests 	[elevio.N_Floors][elevio.N_Buttons - 1]bool,
					StateUpdateCh 			chan hall_request_assigner.ActiveElevator,
					HallOrderCompleteCh 	chan elevio.ButtonEvent,
					DisconnectedElevatorCh 	chan string,
					AssignHallRequestsCh 	chan map[string][elevio.N_Floors][elevio.N_Buttons - 1]bool,
					AckCh 					chan bool,
					ConsumerChannels 		map[net.Conn]chan map[string][elevio.N_Floors][elevio.N_Buttons - 1]bool,
					connChan 				chan net.Conn,
) {

	clientTxEnable := make(chan bool)
	BroadcastCombinedHallRequestsCh := make(chan [elevio.N_Floors][elevio.N_Buttons - 1]bool, 1024)
	CloseConnCh := make(chan string, 1024)

	go UDPCheckPeerAliveStatus(LOCAL_ELEVATOR_ALIVE_PORT, DisconnectedElevatorCh, CloseConnCh)
	go UDPBroadCastPrimaryRole(DETECTION_PORT, clientTxEnable)
	go UDPBroadCastCombinedHallRequests(HALL_LIGHTS_PORT, CombinedHallRequests, BroadcastCombinedHallRequestsCh)                                                                            //Continously broadcast that you are a primary on UDP
	go TCPListenForNewElevators(TCP_LISTEN_PORT, StateUpdateCh, HallOrderCompleteCh, DisconnectedElevatorCh, AssignHallRequestsCh, ConsumerChannels, CloseConnCh, connChan) //Continously listen if new elevator entring networks is trying to establish connection
	go HandlePrimaryTasks(ActiveElevatorMap, CombinedHallRequests, StateUpdateCh, HallOrderCompleteCh, DisconnectedElevatorCh, AssignHallRequestsCh, AckCh, BroadcastCombinedHallRequestsCh)

}

func HandlePrimaryTasks(ActiveElevatorMap 					map[string]elevator.Elevator,
						CombinedHallRequests 				[elevio.N_Floors][elevio.N_Buttons - 1]bool,
						StateUpdateCh 						chan hall_request_assigner.ActiveElevator,
						HallOrderCompleteCh 				chan elevio.ButtonEvent,
						DisconnectedElevatorCh 				chan string,
						AssignHallRequestsCh 				chan map[string][elevio.N_Floors][elevio.N_Buttons - 1]bool,
						AckCh 								chan bool,
						BroadcastCombinedHallRequestsCh 	chan [elevio.N_Floors][elevio.N_Buttons - 1]bool,
) {

	BackupAddr := ""
	var backupConn net.Conn
	// Guarantees that when Backup is promoted to Primary, that the current active hall requests are redistributed without the necessity of an event occuring (i.e. button pressed, floor arrival, elevator disconnected)
	if len(ActiveElevatorMap) > 0 { 
		StateUpdateCh <- hall_request_assigner.ActiveElevator{
			Elevator:  ActiveElevatorMap[GetMapKey(ActiveElevatorMap)],
			MyAddress: GetMapKey(ActiveElevatorMap),
		}
	}
	for {
		// Guarantees that ActiveElevatorMap contains Primary
		if _, exists := ActiveElevatorMap[GetLocalIPv4()]; !exists && len(ActiveElevatorMap) > 0 {
			fmt.Println("Primary is not in ActiveElevatorMap. Adding it.")
			StateUpdateCh <- hall_request_assigner.ActiveElevator{
				Elevator:  ActiveElevatorMap[GetMapKey(ActiveElevatorMap)],
				MyAddress: GetLocalIPv4(),
			}
		}

		select {
		case stateUpdate := <-StateUpdateCh:
			fmt.Println("StateUpdate: ", stateUpdate)
			ActiveElevatorMap[stateUpdate.MyAddress] = stateUpdate.Elevator

			if len(ActiveElevatorMap) >= 2 {
				if _, exists := ActiveElevatorMap[BackupAddr]; !exists {
					fmt.Println("Backup does not exists yet. Initializing it..")
					BackupAddr = GetBackupAddress(ActiveElevatorMap)
					backupConn = TCPDialBackup(BackupAddr, TCP_BACKUP_PORT)
					go TCPReadACK(backupConn, DisconnectedElevatorCh, AckCh) 
					for ip, elev := range ActiveElevatorMap {
						TCPSendActiveElevator(backupConn, hall_request_assigner.ActiveElevator{Elevator: elev, MyAddress: ip})
					}
				}
				TCPSendActiveElevator(backupConn, stateUpdate)
				go func() {
					select {
					case <-AckCh:
						CombinedHallRequests = UpdateCombinedHallRequests(ActiveElevatorMap, CombinedHallRequests)
						BroadcastCombinedHallRequestsCh <- CombinedHallRequests
						AssignHallRequestsCh <- hall_request_assigner.HallRequestAssigner(ActiveElevatorMap, CombinedHallRequests)
					case <-time.After(5 * time.Second):
						fmt.Println("No ACK recieved - Timeout occurred.")
					}
				}()
			}

		case completedOrder := <-HallOrderCompleteCh:
			if len(ActiveElevatorMap) >= 2 {
				if _, exists := ActiveElevatorMap[BackupAddr]; !exists {
					fmt.Println("Backup does not exists yet. Initializing it..")
					BackupAddr = GetBackupAddress(ActiveElevatorMap)
					backupConn = TCPDialBackup(BackupAddr, TCP_BACKUP_PORT)
					go TCPReadACK(backupConn, DisconnectedElevatorCh, AckCh)
					for ip, elev := range ActiveElevatorMap {
						TCPSendActiveElevator(backupConn, hall_request_assigner.ActiveElevator{Elevator: elev, MyAddress: ip})
					}
				}
				TCPSendButtonEvent(backupConn, completedOrder)
				go func() {
					select {
					case <-AckCh:
						CombinedHallRequests[completedOrder.Floor][completedOrder.Button] = false
						BroadcastCombinedHallRequestsCh <- CombinedHallRequests
					case <-time.After(5 * time.Second):
						fmt.Println("No ACK recieved - Timeout occurred.")
					}
				}()
			}

		case disconnectedElevator := <-DisconnectedElevatorCh:
			delete(ActiveElevatorMap, strings.Split(disconnectedElevator, ":")[0])

			if strings.Split(disconnectedElevator, ":")[0] == BackupAddr {
				fmt.Println("Backup disconnected. Removing it ")
				BackupAddr = ""
				backupConn.Close()
			}

			if len(ActiveElevatorMap) >= 2 {
				if _, exists := ActiveElevatorMap[BackupAddr]; !exists {
					fmt.Println("Backup does not exists yet. Initializing it..")
					BackupAddr = GetBackupAddress(ActiveElevatorMap)
					backupConn = TCPDialBackup(BackupAddr, TCP_BACKUP_PORT)
					go TCPReadACK(backupConn, DisconnectedElevatorCh, AckCh) 
					for ip, elev := range ActiveElevatorMap {
						TCPSendActiveElevator(backupConn, hall_request_assigner.ActiveElevator{Elevator: elev, MyAddress: ip})
					}
				}

				TCPSendString(backupConn, disconnectedElevator) // Gives notice to backup of disconnected elevator
				go func() {
					select {
					case <-AckCh:
						AssignHallRequestsCh <- hall_request_assigner.HallRequestAssigner(ActiveElevatorMap, CombinedHallRequests)
					case <-time.After(5 * time.Second):
						fmt.Println("No ACK recieved - Timeout occurred.")
					}
				}()
			}
		}
	}
}

func GetBackupAddress(ActiveElevatorMap map[string]elevator.Elevator) string {
	for key, _ := range ActiveElevatorMap {
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

func TCPSendActiveElevator(conn net.Conn, activeElevator hall_request_assigner.ActiveElevator) error {
	myActiveElevatorMsg := MsgActiveElevator{
		Type:    TypeActiveElevator,
		Content: activeElevator,
	}
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

	data, err := json.Marshal(myMsgCombinedHallRequests) 
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
			fmt.Println("Error reading from connection: ", err)
			continue
		}

		var genericMsg map[string]interface{}
		if err := json.Unmarshal(buf[:n], &genericMsg); err != nil {
			fmt.Println("Error unmarshaling generic message: ", err)
			continue
		}

		switch MessageType(genericMsg["type"].(string)) {
		case TypeCombinedHallRequests:
			var msg MsgCombinedHallRequests
			if err := json.Unmarshal(buf[:n], &msg); err != nil {
				fmt.Println("Error unmarshaling data: ", err)
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

func TCPReadACK(conn net.Conn, DisconnectedElevatorCh chan string, AckCh chan bool) {

	defer conn.Close()

	for {
		var buf [bufSize]byte
		n, err := conn.Read(buf[:])
		if err != nil {
			DisconnectedElevatorCh <- conn.RemoteAddr().String()
			break
		}

		var genericMsg map[string]interface{}
		if err := json.Unmarshal(buf[:n], &genericMsg); err != nil {
			fmt.Println("Error unmarshaling generic message: ", err)
			continue
		}
		switch MessageType(genericMsg["type"].(string)) {
		case TypeACK:
			var msg MsgACK
			if err := json.Unmarshal(buf[:n], &msg); err != nil {
				fmt.Println("Error unmarshaling message: ", err)
				continue
			}
			AckCh <- msg.Content

		default:
			fmt.Println("Unknown message type")
		}
		time.Sleep(50 * time.Millisecond)
	}
}
