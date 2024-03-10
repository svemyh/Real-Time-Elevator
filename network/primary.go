package network

import (
	"context"
	"elevator/conn"
	"elevator/elevator"
	"elevator/elevio"
	"elevator/hall_request_assigner"
	"encoding/json"
	"log"

	"fmt"
	//"log"
	"net"
	//"reflect"
	"time"
)

type Primary struct {
	Ip       string
	lastSeen time.Time
}

type typeTaggedJSON struct {
	TypeId string
	JSON   []byte
}

const bufSize = 1024

/*
func RunPrimaryBackup(necessarychannels...) {
	if AmIPrimary() {
		PrimaryRoutine()
	} else {
		go run TCPListenForNewPrimary() //Checks the event that a backup has become a new primary and wants to establish connection. This go routine should be shut down at some point
		TCPDialPrimary()
		TCPListenForBackupPromotion() //will simply be a net.Listen("TCP", "primaryAdder"). This blocks code until a connection is established
		BackupRoutine()               //Will never pass prev function unless primary has dialed you to become backup
	}
}

*/
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

// creds to network-go module 
func UDPBroadCastPrimaryRole(ctx context.Context, p string, transmitEnable <-chan bool) { //remove ctx

	port := StringPortToInt(p)

	key := "OptimusPrime"


	conn := conn.DialBroadcastUDP(port) // FIX SO THAT ITS COMPATIBLE WITH STRING

	addr, _ := net.ResolveUDPAddr("udp4", fmt.Sprintf("255.255.255.255:%d", port))
	enable := true
	for {

		select {
		case enable = <-transmitEnable:
		case <-time.After(interval):
		}
		if enable {
			conn.WriteTo([]byte(key), addr)
		}
	}
	/*
	
	checkArgs(chans...)
	typeNames := make([]string, len(chans))
	selectCases := make([]reflect.SelectCase, len(typeNames))
	for i, ch := range chans {
		selectCases[i] = reflect.SelectCase{
			Dir:  reflect.SelectRecv,
			Chan: reflect.ValueOf(ch),
		}
		typeNames[i] = reflect.TypeOf(ch).Elem().String()
	}

	port := StringPortToInt(p)
	conn := conn.DialBroadcastUDP(port)
	addr, _ := net.ResolveUDPAddr("udp4", fmt.Sprintf("255.255.255.255:%d", port))
	for {
		chosen, value, _ := reflect.Select(selectCases)
		jsonstr, _ := json.Marshal(value.Interface())
		ttj, _ := json.Marshal(typeTaggedJSON{
			TypeId: typeNames[chosen],
			JSON:   jsonstr,
		})
		if len(ttj) > bufSize {
			panic(fmt.Sprintf(
				"Tried to send a message longer than the buffer size (length: %d, buffer size: %d)\n\t'%s'\n"+
					"Either send smaller packets, or go to network/bcast/bcast.go and increase the buffer size",
				len(ttj), bufSize, string(ttj)))
		}
		conn.WriteTo(ttj, addr)

	} */
	/*
			//def our local address
			laddr, err := net.ResolveUDPAddr("udp", ":0") // Using the zero-port
			if err != nil {
				fmt.Println("Error resolving UDP address:", err)
				return
			}

			//def remote addr
			raddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("10.100.23.191%s", port)) //addressString to actual address(server/)
			if err != nil {
				fmt.Println("Error resolving UDP address:", err)
				return
			}

			//create a connection to raddr through a socket object
			sockConn, err := net.DialUDP("udp", laddr, raddr)
			if err != nil {
				fmt.Println("Error connecting:", err)
				return
			}

			defer sockConn.Close()

			for {
				//select {
				//case <-ctx.Done(): //?
				//	return
				//default:
				message := "OptimusPrime"

				sockConn.Write([]byte(message))
				//_, err := sockConn.Write([]byte(message))
				//fmt.Printf("Broadcasting: %s\n", message)
				//if err != nil {
				//	log.Printf("Error broadcasting primary role: No one is trying to connect!\n")
				//}
				time.Sleep(1 * time.Second)
				//}
			}
		}
	*/
}

const interval = 15 * time.Millisecond
const timeout time.Duration = 500 * time.Millisecond // 500ms

func AmIPrimary(addressString string, peerUpdateCh chan<- ClientUpdate) (bool, string) { 

	//var buf [1024]byte
	//var p ClientUpdate
	//lastSeen := make(map[string]time.Time)
	port := StringPortToInt(addressString)

	conn := conn.DialBroadcastUDP(port)
	conn.SetReadDeadline(time.Now().Add(interval))

	for {
		buffer := make([]byte, 1024)
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
			return false, addr.String()
		}
		// If received message is not "HelloWorld", keep listening until timeout
	}
	/*
	for {
		//updated := false
		n, _, _ := conn.ReadFrom(buf[0:])

		id := string(buf[:n])

		// Adding new connection
		p.New = ""
		if id != "" {
			if _, idExists := lastSeen[id]; !idExists {
				p.New = id
		//		updated = true
			}

			lastSeen[id] = time.Now()
		}

		// Removing dead connection
		p.Lost = make([]string, 0)
		for k, v := range lastSeen {
			if time.Now().Sub(v) > timeout {
		//		updated = true
				p.Lost = append(p.Lost, k)
				delete(lastSeen, k)
			}
		}*/
/*
		// Sending update
		if updated {
			p.Peers = make([]string, 0, len(lastSeen))

			for k, _ := range lastSeen {
				p.Peers = append(p.Peers, k)
			}

			sort.Strings(p.Peers)
			sort.Strings(p.Lost)
			peerUpdateCh <- p
		}
	} */
	/*

		addr, err := net.ResolveUDPAddr("udp", addressString)
		if err != nil {
			log.Printf("Error resolving UDP address: %v\n", err)
			return false, "err"
		}

		conn, err := net.ListenUDP("udp", addr)
		if err != nil {
			log.Printf("Error listening on UDP: %v\n", err)
			return false, "err"
		}
		defer conn.Close()
	*/

	/*conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	buffer := make([]byte, 1024)
	_, primaryAddr, err := conn.ReadFromUDP(buffer)
	if err != nil {
		if netErr, ok := err.(net.Error); ok && netErr.Timeout() { // assert error to net.Error as this type can be checked as a timeout error
			log.Println("No message received, becoming primary...")
			return true, "err"
		}
		log.Printf("Error reading from UDP: %v\n", err)
	}
	log.Println("Received broadcast from primary, remaining as client...")

	fmt.Println("Message 'I'm Primary' recieved from the address: ", primaryAddr.String())
	return false, primaryAddr.String()
	*/

	// Set a deadline for reading. Read operations will fail if no data is received after 5 seconds.
	/*
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))

	


*/
}


func TCPListenForNewElevators(TCPPort string, StateUpdateCh chan hall_request_assigner.ActiveElevator, HallOrderCompleteCh chan elevio.ButtonEvent, DisconnectedElevatorCh chan string, AssignHallRequestsCh chan map[string][elevio.N_Floors][elevio.N_Buttons - 1]bool) {
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

		go TCPReadElevatorStates(conn, StateUpdateCh, HallOrderCompleteCh, DisconnectedElevatorCh)
		go TCPWriteElevatorStates(conn, AssignHallRequestsCh)
		time.Sleep(1 * time.Second)
	}
}

func PrimaryRoutine(StateUpdateCh chan hall_request_assigner.ActiveElevator, HallOrderCompleteCh chan elevio.ButtonEvent, DisconnectedElevatorCh chan string, AssignHallRequestsCh chan map[string][elevio.N_Floors][elevio.N_Buttons - 1]bool) { // Arguments: StateUpdateCh, OrderCompleteCh, ActiveElevators
	//start by establishing TCP connection with yourself (can be done in TCPListenForNewElevators)
	//OR, establish self connection once in RUNPRIMARYBACKUP() and handle selfconnect for future primary in backup.BecomePrimary()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	peerTxEnable := make(chan bool)
	go UDPBroadCastPrimaryRole(ctx, DETECTION_PORT, peerTxEnable)                                                                                //Continously broadcast that you are a primary on UDP
	go TCPListenForNewElevators(TCP_LISTEN_PORT, StateUpdateCh, HallOrderCompleteCh, DisconnectedElevatorCh, AssignHallRequestsCh) //Continously listen if new elevator entring networks is trying to establish connection
	InitActiveElevators := make([]hall_request_assigner.ActiveElevator, 0)
	go HandlePrimaryTasks(StateUpdateCh, HallOrderCompleteCh, InitActiveElevators, DisconnectedElevatorCh, AssignHallRequestsCh)
}

// get new states everytime a local elevator updates their states.
// We then can operate as a primary alone on network or a primary with other elevators on network
// we start by initializing a backup if possible
// then if we have other elevators on network then assign hall req for each elevator(by cost) distribute them and button lights
// if there are other elevators on network then send states to the backup

func HandlePrimaryTasks(StateUpdateCh chan hall_request_assigner.ActiveElevator, HallOrderCompleteCh chan elevio.ButtonEvent, ActiveElevators []hall_request_assigner.ActiveElevator, DisconnectedElevatorCh chan string, AssignHallRequestsCh chan map[string][elevio.N_Floors][elevio.N_Buttons - 1]bool) {
	BackupAddr := ""
	var backupConn net.Conn
	//var ActiveElevators []ActiveElevator // init here or take in as param to func, allows Backup.BecomePrimary to send in prev states
	//can send in as empty array first time primary takes over
	var CombinedHallRequests [elevio.N_Floors][elevio.N_Buttons - 1]bool

	ActiveElevatorMap := make(map[string]elevator.Elevator)

	for {
		select {
		case stateUpdate := <-StateUpdateCh: //updates if new state is sendt on one of TCP conns, blocks if not
			//TODO: compare the state update from single elevator to active elevator array and update activeElevators
			//TODO: update some sort of global HALLREQ array with the new hall requests
			fmt.Println("StateUpdate: ", stateUpdate)

			// 0) If stateUpdate.MyAddress not in ActiveElevators -> stateUpdate ActiveElevators.Append()
			// 1) Overwrite the rest of the states (including cab requests) / 0 & 1 effectively same
			ActiveElevatorMap[stateUpdate.MyAddress] = stateUpdate.Elevator
			//case len(ActiveElevatorMap) == 1:

			// 2) OR new button-presses in new ActiveElevators to CombinedHallRequests // TODO:
			// hall_request_assigner.ActiveElevators_to_CombinedHallRequests() or similar

			// Pick one of the elevators in ActiveElevatorMap

			// 3) IF not backup -> init a backup; IF backup exists -> send stateInfo to backup
			if len(ActiveElevatorMap) >= 2 {
				if _, exists := ActiveElevatorMap[BackupAddr]; !exists {
					// init a backup here:
					fmt.Println("Backup does not exists yet. Initializing it..")
					BackupAddr = GetBackupAddress(ActiveElevatorMap)
					backupConn = TCPDialBackup(BackupAddr, TCP_BACKUP_PORT)
				}
				TCPSendActiveElevator(backupConn, stateUpdate) // Sends recieved ActiveElevator to Backup.
				CombinedHallRequests = UpdateCombinedHallRequests(ActiveElevatorMap, CombinedHallRequests)
				AssignHallRequestsCh <- hall_request_assigner.HallRequestAssigner(ActiveElevatorMap, CombinedHallRequests)

			}

			//for test purpose
			fmt.Println("ØØØØØØØØØØØØØØØØØØØØØØ")
			CombinedHallRequests = UpdateCombinedHallRequests(ActiveElevatorMap, CombinedHallRequests)
			AssignHallRequestsCh <- hall_request_assigner.HallRequestAssigner(ActiveElevatorMap, CombinedHallRequests)
			fmt.Println("ØØØØØØØØØØ - Combinded -ØØØØØØØØØØØØ", CombinedHallRequests)

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

		case CompletedOrder := <-HallOrderCompleteCh:
			//TODO: clear order from some sort of global HALLREQ array
			fmt.Println("\n---- Order completed at floor:", CompletedOrder)
			CombinedHallRequests[CompletedOrder.Floor][CompletedOrder.Button] = false

			if len(ActiveElevatorMap) >= 2 {
				if _, exists := ActiveElevatorMap[BackupAddr]; !exists {
					// init a backup here:
					fmt.Println("Backup does not exists yet. Initializing it..")
					BackupAddr = GetBackupAddress(ActiveElevatorMap)
					backupConn = TCPDialBackup(BackupAddr, TCP_BACKUP_PORT)
				}
				fmt.Println("Break1")
				TCPSendButtonEvent(backupConn, CompletedOrder) // Writing to Backup
				fmt.Println("Break2")
			}

			CombinedHallRequests = UpdateCombinedHallRequests(ActiveElevatorMap, CombinedHallRequests)
			AssignHallRequestsCh <- hall_request_assigner.HallRequestAssigner(ActiveElevatorMap, CombinedHallRequests)

		case disconnectedElevator := <-DisconnectedElevatorCh:
			delete(ActiveElevatorMap, disconnectedElevator)
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

func TCPSendActiveElevator(conn net.Conn, activeElevator hall_request_assigner.ActiveElevator) {
	time.Sleep(50 * time.Millisecond)
	my_ActiveElevatorMsg := MsgActiveElevator{
		Type:    TypeActiveElevator,
		Content: activeElevator,
	}
	fmt.Println("my_ActiveElevatorMsg:", my_ActiveElevatorMsg)
	data, err := json.Marshal(my_ActiveElevatorMsg)

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
	my_ButtonEventMsg := MsgButtonEvent{
		Type:    TypeButtonEvent,
		Content: buttonEvent,
	}

	data, err := json.Marshal(my_ButtonEventMsg)
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

/*


 */
