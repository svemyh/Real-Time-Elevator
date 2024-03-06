package network

import (
	"context"
	"fmt"
	"log"
	"net"
	"time"
)

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

func UDPBroadCastPrimaryRole(port string) {
	//def our local address
	laddr, err := net.ResolveUDPAddr("udp", ":10000") // localIP
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

		//Create an empty buffer to to filled
		buffer := make([]byte, 1024)
		//oob := make([]byte, 1024)

		n := copy(buffer, []byte("Im Primary!"))

		fmt.Printf("copied %d bytes to the buffer from primary: %s\n", n, buffer[:n])

		n, err = sockConn.Write(buffer)

		if err != nil {
			fmt.Println("Error resolving WriteMsgUDP: No one is listening")
		}
		time.Sleep(1 * time.Second)
	}
}

func AmIPrimary() bool {
	currentRole := false // Start as a slave.

	receiverChan := make(chan string)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go InitReceiver(ctx, receiverChan, ":20017")

	messageReceived := false
	timer := time.NewTimer(3 * time.Second) // set to 3 seconds
	for {
		select {
		case msg := <-receiverChan:
			log.Println("Received message:", msg)
			messageReceived = true
			return currentRole
		case <-timer.C:
			if !messageReceived {
				// No message was received within the time frame.
				log.Println("No message received, becoming master...")
				currentRole = false
				cancel() // Stop the Receiver goroutine.
				return currentRole
			}
		case <-ctx.Done():
			return currentRole
		}
		if currentRole {
			break
		}
		messageReceived = false
	}
	return currentRole
}

/*
 */
func PrimaryRoutine() { // Arguments: StateUpdateCh, OrderCompleteCh, ActiveElevators
	//start by establishing TCP connection with yourself (can be done in TCPListenForNewElevators)
	//OR, establish self connection once in RUNPRIMARYBACKUP() and handle selfconnect for future primary in backup.BecomePrimary()
	fmt.Println("PrimaryRoutine")
	go UDPBroadCastPrimaryRole(detectionPort) //Continously broadcast that you are a primary on UDP
	//go run TCPListenForNewElevators() //Continously listen if new elevator entring networks is trying to establish connection
	//go run HandlePrimaryTasks(StateUpdateCh, OrderCompleteCh, ActiveElevators)
}

/*
get new states everytime a local elevator updates their states.
We then can operate as a primary alone on network or a primary with other elevators on network
we start by initializing a backup if possible
then if we have other elevators on network then assign hall req for each elevator(by cost) distribute them and button lights
if there are other elevators on network then send states to the backup


func HandlePrimaryTasks(StateUpdateCh, OrderCompleteCh, ActiveElevators) {
	var backup Elevator
	//var ActiveElevators []Elevator init here or take in as param to func, allows Backup.BecomePrimary to send in prev states
	//can send in as empty array first time primary takes over
	for {
		select {
		case stateUpdate := <- StateUpdateCh:					//updates if new state is sendt on one of TCP conns, blocks if not
			//TODO: compare the state update from single elevator to active elevator array and update activeElevators
			//TODO: update some sort of global HALLREQ array with the new hall requests

			//init a backup
			if backup == nil {
				if len(ActiveElevators) > 1 {
					//init a backup
				}
			}

			if  len(ActiveElevators) > 1 {
				//TODO: assign  new backup if needed based based on state update.
				//TODO: send updated states to backup (with ack) (if there still is a backup)
				//TODO: assign new new hall orders for each elevator through cost-func
				DistributeHallRequests(assignedHallReq)			   //Distribute new hall requests to each elevator, needs ack and blocking until done
				DistributeHallButtonLights(assignedHallReq)		   //Distribute the button lights to each now that we have ack from each
			} else {
				//TODO: assign new new hall orders for each elevator through cost-func. we are now a primary alone on network
				//TODO: Should have some check to see if are a primary that lost network (so a new primary has been made) or if we have network connection and no other elevators on net
				backup = nil
				DistributeHallRequests()			   //Distribute new hall requests to each elevator, needs ack and blocking until done
				DistributeHallButtonLights()		   //Distribute the button lights to each now that we have ack from each
			}
		case CompletedOrder := <- OrderCompleteCh:
			//TODO: clear order from some sort of global HALLREQ array
		}
	}
}
*/
