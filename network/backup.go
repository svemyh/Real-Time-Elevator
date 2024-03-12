package network

import (
	"context"
	"elevator/conn"
	"elevator/elevator"
	"elevator/elevio"
	"elevator/hall_request_assigner"
	"fmt"
	"log"
	"net"
	"time"
)

type Backup struct {
	Ip       string
	lastSeen time.Time
}

func BackupRoutine(conn net.Conn,
	primaryAddress string,
	FSMStateUpdateCh chan hall_request_assigner.ActiveElevator,
	FSMHallOrderCompleteCh chan elevio.ButtonEvent,
	FSMAssignedHallRequestsCh chan [elevio.N_Floors][elevio.N_Buttons - 1]bool) {

	BackupActiveElevatorMap := make(map[string]elevator.Elevator)
	var BackupCombinedHallRequests [elevio.N_Floors][elevio.N_Buttons - 1]bool

	//TODO: Monitor that primary connection is alive
	//TODO: Read states sendt through primary connection (make an array to contain -> activeElevators)
	//TODO: If backup unresponsive --> BecomePrimary(activeElevators). TODO: INIT A BACKUP IN BecomePrimary()
	//fmt.Println("Im a backup, doing backup things")
	PrimaryDeadCh := make(chan bool)

	backupCtx, cancelBackup := context.WithCancel(ctx)
	defer cancelBackup()

	go TCPListenForNewPrimary(backupCtx, TCP_LISTEN_PORT, FSMStateUpdateCh, FSMHallOrderCompleteCh, FSMAssignedHallRequestsCh)

	go CheckPrimaryAlive(primaryAddress, PrimaryDeadCh)

	BackupStateUpdateCh := make(chan hall_request_assigner.ActiveElevator)
	BackupHallOrderCompleteCh := make(chan elevio.ButtonEvent)
	BackupDisconnectedElevatorCh := make(chan string)

	go TCPReadElevatorStates(conn, BackupStateUpdateCh, BackupHallOrderCompleteCh, BackupDisconnectedElevatorCh)

	for {
		select {
		case stateUpdate := <-BackupStateUpdateCh:
			fmt.Println("BACKUP received stateUpdate: ", stateUpdate)
			BackupActiveElevatorMap[stateUpdate.MyAddress] = stateUpdate.Elevator
			BackupCombinedHallRequests = UpdateCombinedHallRequests(BackupActiveElevatorMap, BackupCombinedHallRequests)
			TCPSendACK(conn)

		case completedOrder := <-BackupHallOrderCompleteCh:
			fmt.Println("BACKUP received completedOrder: ", completedOrder)
			BackupCombinedHallRequests[completedOrder.Floor][completedOrder.Button] = false
			TCPSendACK(conn)

		case disconnectedElevator := <-BackupDisconnectedElevatorCh:
			fmt.Println("BACKUP received disconnectedElevator: ", disconnectedElevator)
			delete(BackupActiveElevatorMap, disconnectedElevator)
			TCPSendACK(conn)

		case <-PrimaryDeadCh:
			log.Printf("Becoming Primary")
			cancelBackup()
			fmt.Println("Requesting cancellation")	// why doesnt TCPListenForNewPrimary cancel prior to time.Sleep independent of time?
			time.Sleep(1 * time.Second)
			go PrimaryRoutine(true, NewElevatorSystemChannels().StateUpdateCh, NewElevatorSystemChannels().HallOrderCompleteCh, NewElevatorSystemChannels().DisconnectedElevatorCh, NewElevatorSystemChannels().AssignHallRequestsMapCh, NewElevatorSystemChannels().AckCh)
			time.Sleep(1500 * time.Millisecond)
			TCPDialPrimary(GetLocalIPv4()+TCP_LISTEN_PORT, FSMStateUpdateCh, FSMHallOrderCompleteCh, FSMAssignedHallRequestsCh)
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

		message := string(buffer[:n])
		if message == "OptimusPrime" {
			log.Printf("Received %s from primary, remaining as backup...", message)
			conn.SetReadDeadline(time.Now().Add(udpInterval)) // Reset readDeadline
		}
		// If received message is not "OptimusPrime", keep listening until timeout
		//At timeout become a primary
	}
}

/*
func BecomePrimary(activeElevator []Elevator) {
	//TODO: ALl below
	//establish TCP connection with all elevators last primary had connections with as a client
	//has to get a TCP conn object with all new elevators to be able to interact in PrimaryRoutine
	//Needs to run a goroutine TCPReadElevatorStates when connection established
	//Note, the elevator list will contain your own IP. DO NOT CONNECT TO IT, as PrimaryRoutine will make this connection
	//(OR HANDLE HERE IF DESIRED)
	PrimaryRoutine(activeElevators)
}
*/

func BackupReceiver(TCPPort string) {

}

func BackupTransmitter(TCPPort string) {

}
