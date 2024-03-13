package network

import (
	"elevator/conn"
	"elevator/elevator"
	"elevator/elevio"
	"elevator/hall_request_assigner"
	"fmt"
	"log"
	"net"
	"strings"
	"time"
)

func BackupRoutine(conn net.Conn, primaryAddress string, E ElevatorSystemChannels) {
	BackupActiveElevatorMap := make(map[string]elevator.Elevator)
	var BackupCombinedHallRequests [elevio.N_Floors][elevio.N_Buttons - 1]bool

	//TODO: Monitor that primary connection is alive
	//TODO: Read states sendt through primary connection (make an array to contain -> activeElevators)
	//TODO: If backup unresponsive --> BecomePrimary(activeElevators). TODO: INIT A BACKUP IN BecomePrimary()
	PrimaryDeadCh := make(chan bool)
	go CheckPrimaryAlive(primaryAddress, PrimaryDeadCh)

	BackupCh := ElevatorSystemChannels{
		StateUpdateCh: 			make(chan hall_request_assigner.ActiveElevator),
		HallOrderCompleteCh:		make(chan elevio.ButtonEvent),
		DisconnectedElevatorCh:	make(chan string),
	}

	go TCPReadElevatorStates(conn, BackupCh)

	for {
		select {
		case stateUpdate := <-BackupCh.StateUpdateCh:
			fmt.Println("BACKUP recieved stateUpdate: ", stateUpdate)
			BackupActiveElevatorMap[stateUpdate.MyAddress] = stateUpdate.Elevator
			BackupCombinedHallRequests = UpdateCombinedHallRequests(BackupActiveElevatorMap, BackupCombinedHallRequests)
			TCPSendACK(conn)

		case completedOrder := <-BackupCh.HallOrderCompleteCh:
			fmt.Println("BACKUP recieved completedOrder: ", completedOrder)
			BackupCombinedHallRequests[completedOrder.Floor][completedOrder.Button] = false
			TCPSendACK(conn)

		case disconnectedElevator := <-BackupCh.DisconnectedElevatorCh:
			fmt.Println("BACKUP recieved disconnectedElevator: ", disconnectedElevator)
			delete(BackupActiveElevatorMap, disconnectedElevator)
			TCPSendACK(conn)
		case <-PrimaryDeadCh:
			log.Println("Primary confirmed dead")
			delete(BackupActiveElevatorMap, strings.Split(primaryAddress, ":")[0])
			time.Sleep(1 * time.Second)
			BecomeNewPrimary(BackupActiveElevatorMap, BackupCombinedHallRequests, E)

		}
	}
}

// Read "I'm the Primary" -message from the Primary(). If no message is recieved after N seconds,
// then BackupRoutine() can assume Primary is dead -> Promote itself to Primary.
func CheckPrimaryAlive(primaryAddress string, PrimaryDeadCh chan bool) {
	//addr, err := net.ResolveUDPAddr("udp", primaryAddress)
	//if err != nil {
	//	log.Printf("-CheckPrimaryAlive() Error resolving UDP address: %v\n", err)
	//	return
	//}

	conn := conn.DialBroadcastUDP(StringPortToInt(DETECTION_PORT))

	conn.SetReadDeadline(time.Now().Add(udpInterval))
	for {
		buffer := make([]byte, bufSize)
		n, _, err := conn.ReadFrom(buffer)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				log.Printf("Timeout reached without receiving %s, backup is becoming primary...", buffer[:n])
				PrimaryDeadCh <- true
				return
			}
			log.Printf("Error reading from UDP: %v\n", err)
			return
		}
		// If received message is not "OptimusPrime", keep listening until timeout
		//At timeout become a primary
		message := string(buffer[:n])
		if message == "OptimusPrime" {
			log.Printf("Received %s from primary, remaining as backup...", message)
			conn.SetReadDeadline(time.Now().Add(udpInterval)) // Reset readDeadline
		}
	}
}

func BecomeNewPrimary(BackupActiveElevatorMap map[string]elevator.Elevator,
	BackupCombinedHallRequests [elevio.N_Floors][elevio.N_Buttons - 1]bool,
	E ElevatorSystemChannels) {
	//TODO: ALl below
	//establish TCP connection with all elevators last primary had connections with as a client
	//has to get a TCP conn object with all new elevators to be able to interact in PrimaryRoutine
	//Needs to run a goroutine TCPReadElevatorStates when connection established
	//Note, the elevator list will contain your own IP. DO NOT CONNECT TO IT, as PrimaryRoutine will make this connection
	//(OR HANDLE HERE IF DESIRED)

	//TCPDialAsPrimary
	for ip := range BackupActiveElevatorMap {
		fmt.Println("Connecting by TCP to the address: ", ip+TCP_NEW_PRIMARY_LISTEN_PORT)

		conn, err := net.Dial("tcp", ip+TCP_NEW_PRIMARY_LISTEN_PORT)
		if err != nil {
			fmt.Println("Connection failed. Error: ", err)
			return
		}

		fmt.Println("Conection established to: ", conn.RemoteAddr())

		go TCPReadElevatorStates(conn, E)
		go TCPWriteElevatorStates(conn, E)
	}

	NewPrimaryChannels := ElevatorSystemChannels{
		BackupActiveElevatorMap,
		BackupCombinedHallRequests,
		E.StateUpdateCh,
		E.HallOrderCompleteCh,
		E.DisconnectedElevatorCh,
		E.AssignHallRequestsMapCh,
		E.AckCh,
	}

	time.Sleep(1500 * time.Millisecond)

/*
	PrimaryRoutine(BackupActiveElevatorMap, BackupCombinedHallRequests, StateUpdateCh, HallOrderCompleteCh, DisconnectedElevatorCh, AssignHallRequestsCh, AckCh)

*/
	PrimaryRoutine(NewPrimaryChannels)

}
