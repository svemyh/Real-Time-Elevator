package network

import (
	"context"
	"elevator/conn"
	"fmt"
	"log"
	"net"
	"time"
)

type Backup struct {
	Ip       string
	lastSeen time.Time
}

func BackupRoutine(conn net.Conn, primaryAddress string) {

	//TODO: Monitor that primary connection is alive
	//TODO: Read states sendt through primary connection (make an array to contain -> activeElevators)
	//TODO: If backup unresponsive --> BecomePrimary(activeElevators)
	fmt.Println("Im a backup, doing backup things")
	PrimaryDeadCh := make(chan bool)
	go CheckPrimaryAlive(primaryAddress, PrimaryDeadCh)
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

func BackupReceiver(ctx context.Context, TCPPort string) {

}

func BackupTransmitter(TCPPort string) {

}
