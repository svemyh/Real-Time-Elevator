package primary_backup

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

func BackupRoutine(conn 					net.Conn,
				   StateUpdateCh 			chan hall_request_assigner.ActiveElevator,
				   HallOrderCompleteCh 		chan elevio.ButtonEvent,
				   DisconnectedElevatorCh 	chan string,
				   AssignHallRequestsCh 	chan map[string][elevio.N_Floors][elevio.N_Buttons - 1]bool,
				   AckCh 					chan bool,
) {
	BackupActiveElevatorMap := make(map[string]elevator.Elevator)
	var BackupCombinedHallRequests [elevio.N_Floors][elevio.N_Buttons - 1]bool

	fmt.Println("I am now a Backup")

	PrimaryDeadCh := make(chan bool)
	go CheckPrimaryAlive(PrimaryDeadCh)

	BackupStateUpdateCh := make(chan hall_request_assigner.ActiveElevator, bufSize)
	BackupHallOrderCompleteCh := make(chan elevio.ButtonEvent, bufSize)
	BackupDisconnectedElevatorCh := make(chan string, bufSize)

	go TCPReadElevatorStates(conn, BackupStateUpdateCh, BackupHallOrderCompleteCh, BackupDisconnectedElevatorCh)

	for {
		select {
		case stateUpdate := <-BackupStateUpdateCh:
			BackupActiveElevatorMap[stateUpdate.MyAddress] = stateUpdate.Elevator
			BackupCombinedHallRequests = UpdateCombinedHallRequests(BackupActiveElevatorMap, BackupCombinedHallRequests)
			TCPSendACK(conn)

		case completedOrder := <-BackupHallOrderCompleteCh:
			BackupCombinedHallRequests[completedOrder.Floor][completedOrder.Button] = false
			TCPSendACK(conn)

		case disconnectedElevator := <-BackupDisconnectedElevatorCh:
			delete(BackupActiveElevatorMap, disconnectedElevator)
			TCPSendACK(conn)
		case <-PrimaryDeadCh:
			delete(BackupActiveElevatorMap, strings.Split(conn.RemoteAddr().String(), ":")[0])
			fmt.Println("Primary confirmed dead with the address: ", strings.Split(conn.RemoteAddr().String(), ":")[0])
			time.Sleep(1 * time.Second)
			BecomePrimary(BackupActiveElevatorMap, BackupCombinedHallRequests, StateUpdateCh, HallOrderCompleteCh, DisconnectedElevatorCh, AssignHallRequestsCh, AckCh)

		}
	}
}

func CheckPrimaryAlive(PrimaryDeadCh chan bool) {

	conn := conn.DialBroadcastUDP(StringPortToInt(DETECTION_PORT))

	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
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
		if message == "I'm the Primary" {
			fmt.Printf("Received %s from primary, remaining as backup...", message)
			conn.SetReadDeadline(time.Now().Add(udpInterval))
		}
	}
}

func BecomePrimary(BackupActiveElevatorMap 		map[string]elevator.Elevator,
				   BackupCombinedHallRequests 	[elevio.N_Floors][elevio.N_Buttons - 1]bool,
				   StateUpdateCh 				chan hall_request_assigner.ActiveElevator,
				   HallOrderCompleteCh 			chan elevio.ButtonEvent,
				   DisconnectedElevatorCh 		chan string,
				   AssignHallRequestsCh 		chan map[string][elevio.N_Floors][elevio.N_Buttons - 1]bool,
				   AckCh 						chan bool,
) {

	// When AssignedHallRequestsCh recieves a message, StartBroadcaster() distributes it to each of the personalAssignedHallRequestsCh used in TCPWriteElevatorStates()
	ConsumerChannels := make(map[net.Conn]chan map[string][elevio.N_Floors][elevio.N_Buttons - 1]bool)
	ConnChan := make(chan net.Conn, 1024)

	fmt.Println("BackupActiveElevatorMap: ", BackupActiveElevatorMap)
	for ip, _ := range BackupActiveElevatorMap {
		fmt.Println("Connecting by TCP to the address: ", ip+TCP_NEW_PRIMARY_LISTEN_PORT)

		conn, err := net.Dial("tcp", ip+TCP_NEW_PRIMARY_LISTEN_PORT)
		if err != nil {
			fmt.Println("Connection failed. Error: ", err)
			return
		}

		fmt.Println("Conection established to: ", conn.RemoteAddr())
		personalAssignedHallRequestsCh := make(chan map[string][elevio.N_Floors][elevio.N_Buttons - 1]bool)
		ConsumerChannels[conn] = personalAssignedHallRequestsCh
		ConnChan <- conn
		go TCPWriteElevatorStates(conn, personalAssignedHallRequestsCh)

		go TCPReadElevatorStates(conn, StateUpdateCh, HallOrderCompleteCh, DisconnectedElevatorCh)

	}

	time.Sleep(1500 * time.Millisecond)

	PrimaryRoutine(BackupActiveElevatorMap, BackupCombinedHallRequests, StateUpdateCh, HallOrderCompleteCh, DisconnectedElevatorCh, AssignHallRequestsCh, AckCh, ConsumerChannels, ConnChan)
}
