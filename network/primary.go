package network

import (
	"context"
	"elevator/elevator"
	"elevator/elevio"
	"elevator/hall_request_assigner"
	"fmt"
	"log"
	"net"
	"time"
)

type Primary struct {
	Ip       string
	lastSeen time.Time
}

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

func UDPBroadCastPrimaryRole(ctx context.Context, port string) {
	//def our local address
	laddr, err := net.ResolveUDPAddr("udp", ":0") // Using the zero-port
	if err != nil {
		fmt.Println("Error resolving UDP address:", err)
		return
	}

	//def remote addr
	raddr, err := net.ResolveUDPAddr("udp", port) //addressString to actual address(server/)
	if err != nil {
		fmt.Println("Error resolving UDP address:", err)
		return
	}

	//create a connection to raddr through a socket object
	sockConn, err := net.DialUDP("udp", laddr, raddr)

	defer sockConn.Close()

	for {
		select {
		case <-ctx.Done(): //?
			return
		default:
			message := "OptimusPrime"

			sockConn.Write([]byte(message))
			//_, err := sockConn.Write([]byte(message))
			//fmt.Printf("Broadcasting: %s\n", message)
			//if err != nil {
			//	log.Printf("Error broadcasting primary role: No one is trying to connect!\n")
			//}
			time.Sleep(1 * time.Second)
		}
	}
}

func AmIPrimary(addressString string) (bool, string) {
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
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))

	for {
		buffer := make([]byte, 1024)
		n, addr, err := conn.ReadFromUDP(buffer)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				log.Println("Timeout reached without receiving 'OptimusPrime', becoming primary...")
				return true, "none"
			}
			log.Printf("Error reading from UDP: %v\n", err)
			return false, "err"
		}

		message := string(buffer[:n])
		if message == "OptimusPrime" {
			log.Println("Received 'OptimusPrime' from primary, remaining as client...")
			return false, addr.String()
		}
		// If received message is not "HelloWorld", keep listening until timeout
	}
}

func TCPListenForNewElevators(TCPPort string, StateUpdateCh chan hall_request_assigner.ActiveElevator, HallOrderCompleteCh chan elevio.ButtonEvent, DisconnectedElevatorCh chan string) {
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
		time.Sleep(1 * time.Second)
	}
}

func PrimaryRoutine(StateUpdateCh chan hall_request_assigner.ActiveElevator, HallOrderCompleteCh chan elevio.ButtonEvent, DisconnectedElevatorCh chan string) { // Arguments: StateUpdateCh, OrderCompleteCh, ActiveElevators
	//start by establishing TCP connection with yourself (can be done in TCPListenForNewElevators)
	//OR, establish self connection once in RUNPRIMARYBACKUP() and handle selfconnect for future primary in backup.BecomePrimary()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go UDPBroadCastPrimaryRole(ctx, DETECTION_PORT)                                                          //Continously broadcast that you are a primary on UDP
	go TCPListenForNewElevators(TCP_LISTEN_PORT, StateUpdateCh, HallOrderCompleteCh, DisconnectedElevatorCh) //Continously listen if new elevator entring networks is trying to establish connection
	InitActiveElevators := make([]hall_request_assigner.ActiveElevator, 0)
	go HandlePrimaryTasks(StateUpdateCh, HallOrderCompleteCh, InitActiveElevators, DisconnectedElevatorCh)
}

// get new states everytime a local elevator updates their states.
// We then can operate as a primary alone on network or a primary with other elevators on network
// we start by initializing a backup if possible
// then if we have other elevators on network then assign hall req for each elevator(by cost) distribute them and button lights
// if there are other elevators on network then send states to the backup

func HandlePrimaryTasks(StateUpdateCh chan hall_request_assigner.ActiveElevator, HallOrderCompleteCh chan elevio.ButtonEvent, ActiveElevators []hall_request_assigner.ActiveElevator, DisconnectedElevatorCh chan string) {
	BackupAddr := ""
	var backupConn net.Conn
	//var ActiveElevators []ActiveElevator // init here or take in as param to func, allows Backup.BecomePrimary to send in prev states
	//can send in as empty array first time primary takes over
	var CombinedHallRequests [elevio.N_Floors][elevio.N_Buttons - 1]bool
	var ActiveElevatorMap map[string]elevator.Elevator

	for {
		select {
		case stateUpdate := <-StateUpdateCh: //updates if new state is sendt on one of TCP conns, blocks if not
			//TODO: compare the state update from single elevator to active elevator array and update activeElevators
			//TODO: update some sort of global HALLREQ array with the new hall requests
			fmt.Println("StateUpdate: ", stateUpdate)

			// 0) If stateUpdate.MyAddress not in ActiveElevators -> stateUpdate ActiveElevators.Append()
			// 1) Overwrite the rest of the states (including cab requests) / 0 & 1 effectively same
			ActiveElevatorMap[stateUpdate.MyAddress] = stateUpdate.Elevator

			// 2) OR new button-presses in new ActiveElevators to CombinedHallRequests // TODO:
			// hall_request_assigner.ActiveElevators_to_CombinedHallRequests() or similar

			// 3) IF not backup -> init a backup; IF backup exists -> send stateInfo to backup
			if _, exists := ActiveElevatorMap[BackupAddr]; !exists {
				fmt.Println("Backup does not exists yet. Initializing it..")
				backupConn = TCPDialBackup(BackupAddr, TCP_BACKUP_PORT)
			}
			//sendLocalStatesToPrimaryLoop(backupConn) // something similar but not this
			fmt.Println(backupConn.RemoteAddr().String())

			//---NEED NOT BECAUSE IGNORE CASE OF N ELEVATORS = 1
			// if len(ActiveElevators) ==1 { // (Primary is alone)
			// do nothing
			//}
			//-----------------------------------------------

			// if BackupAddr not in ActiveElevators {
			//		conn = DialBackup(BackupAddr)
			// }
			//
			// WriteStatesToBackup(stateUpdate)
			// Wait for ack
			//

			if len(ActiveElevators) > 1 {
				//TODO: assign  new backup if needed based based on state update.
				//TODO: send updated states to backup (with ack) (if there still is a backup)
				//TODO: assign new new hall orders for each elevator through cost-func

				//DistributeHallRequests(assignedHallReq)     //Distribute new hall requests to each elevator, needs ack and blocking until done
				//DistributeHallButtonLights(assignedHallReq) //Distribute the button lights to each now that we have ack from each
			} else {
				//TODO: assign new new hall orders for each elevator through cost-func. we are now a primary alone on network
				//TODO: Should have some check to see if are a primary that lost network (so a new primary has been made) or if we have network connection and no other elevators on net

				//backup = nil
				//DistributeHallRequests()     //Distribute new hall requests to each elevator, needs ack and blocking until done
				//DistributeHallButtonLights() //Distribute the button lights to each now that we have ack from each
			}
		case CompletedOrder := <-HallOrderCompleteCh:
			//TODO: clear order from some sort of global HALLREQ array
			fmt.Println("\n---- Order completed at floor:", CompletedOrder)
			CombinedHallRequests[CompletedOrder.Floor][CompletedOrder.Button] = false
			// WriteStatesToBackup(CombinedHallRequests)
		case disconnectedElevator := <-DisconnectedElevatorCh:
			delete(ActiveElevatorMap, disconnectedElevator)
		}
	}
}

/*


 */
