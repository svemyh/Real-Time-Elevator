package network

import (
	"elevator/elevio"
	"elevator/hall_request_assigner"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

func NewElevatorSystemChannels() ElevatorSystemChannels {
	return ElevatorSystemChannels{
		//ActiveElevatorMap 		make(map[string]elevator.Elevator)
		//CombinedHallRequests 	  [elevio.N_Floors][elevio.N_Buttons-1]bool
		StateUpdateCh:             make(chan hall_request_assigner.ActiveElevator, 1024),
		HallOrderCompleteCh:       make(chan elevio.ButtonEvent, 1024),
		DisconnectedElevatorCh:    make(chan string, 1024),
		AssignHallRequestsMapCh:   make(chan map[string][elevio.N_Floors][elevio.N_Buttons - 1]bool, 1024),
		AckCh:                     make(chan bool, 64),
	}
}

func NewFSMSystemChannels() FSMSystemChannels {
	return FSMSystemChannels{
		FSMStateUpdateCh:          make(chan hall_request_assigner.ActiveElevator, 1024),
		FSMHallOrderCompleteCh:    make(chan elevio.ButtonEvent, 1024),
		FSMAssignedHallRequestsCh: make(chan [elevio.N_Floors][elevio.N_Buttons - 1]bool),
	}
}

// Alias: RunPrimaryBackup()
func InitNetwork(F FSMSystemChannels, E ElevatorSystemChannels) {
	clientUpdateCh := make(chan ClientUpdate)
	//clientTxEnable := make(chan bool)
	var id string

	isPrimary, primaryAddress := AmIPrimary(DETECTION_PORT, clientUpdateCh)
	if isPrimary {
		if id == "" {
			localIP, err := LocalIP()
			if err != nil {
				fmt.Println(err)
				localIP = "DISCONNECTED"
			}
			id = fmt.Sprintf("Master-%s-%d", localIP, os.Getpid())
			fmt.Printf("My id: %s\n", id)
		}

		log.Println("Operating as primary...")
		go PrimaryRoutine(E)
		time.Sleep(1500 * time.Millisecond)
		TCPDialPrimary(GetLocalIPv4()+TCP_LISTEN_PORT, F)
	} else {
		if id == "" {
			localIP, err := LocalIP()
			if err != nil {
				fmt.Println(err)
				localIP = "DISCONNECTED"
			}
			id = fmt.Sprintf("Client-%s-%d\n", localIP, os.Getpid())
			fmt.Printf("My id: %s", id)
		}
		log.Println("Operating as client...")
		go TCPDialPrimary(primaryAddress+TCP_LISTEN_PORT, F)
		go TCPListenForNewPrimary(TCP_NEW_PRIMARY_LISTEN_PORT, F)
		conn, _ := TCPListenForBackupPromotion(TCP_BACKUP_PORT) //will simply be a net.Listen("TCP", "primaryAdder"). This blocks code until a connection is established
		BackupRoutine(conn, primaryAddress+DETECTION_PORT, E)
	}
}

// Checks the event that a backup has become a new primary and wants to establish connection. This go routine should be shut down at some point
func TCPListenForNewPrimary(TCPPort string, F FSMSystemChannels) {
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

		go RecieveAssignedHallRequests(conn, F)
		go sendLocalStatesToPrimaryLoop(conn, F) // This will terminate whenever the connection/conn is closed - i.e. conn.Write() throws an error.
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

func TCPDialPrimary(PrimaryAddress string, F FSMSystemChannels) {
	fmt.Println("Connecting by TCP to the address: ", PrimaryAddress)

	conn, err := net.Dial("tcp", PrimaryAddress)
	if err != nil {
		fmt.Println("Connection failed. Error: ", err)
		return
	}

	fmt.Println("Conection established to: ", conn.RemoteAddr())
	defer conn.Close()

	go RecieveAssignedHallRequests(conn, F)
	sendLocalStatesToPrimaryLoop(conn, F)
}

func RecieveAssignedHallRequests(conn net.Conn, F FSMSystemChannels) { // NOT TESTED!
	fmt.Printf("RecieveAssignedHallRequests() - *New connection accepted from address: %s\n", conn.LocalAddr())

	defer conn.Close()

	for {
		var buf [bufSize]byte
		n, err := conn.Read(buf[:])
		if err != nil {
			// Error means TCP-conn has broken -> TODO: Do something
			println("OH NO, The conn at line 224 broke1: %e", err)
			//log.Fatal(err)
			break
		}

		// Unmarshal JSON data into a map of elevator states
		var assignedHallRequests [elevio.N_Floors][elevio.N_Buttons - 1]bool
		err = json.Unmarshal(buf[:n], &assignedHallRequests)
		if err != nil {
			//return err
			println("OH NO, The conn at line 224 broke2: %e", err)
			//log.Fatal(err)
			break
		}
		F.FSMAssignedHallRequestsCh <- assignedHallRequests
		time.Sleep(1 * time.Second)
	}
}

func sendLocalStatesToPrimaryLoop(conn net.Conn, F FSMSystemChannels) {
	fmt.Println("- sendLocalStatesToPrimaryLoop() - Conection established to: ", conn.RemoteAddr())
	defer conn.Close()

	for {
		select {
		case stateUpdate := <-F.FSMStateUpdateCh:
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
			TCPSendActiveElevator(conn, stateUpdate)

		case hallOrderComplete := <-F.FSMHallOrderCompleteCh:
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
	}
}

//receiverChan := make(chan string)
//go network.Reciever(receiverChan, "localhost:20013")

func TCPWriteElevatorStates(conn net.Conn, E ElevatorSystemChannels) {
	defer conn.Close()

	for {
		assignedHallRequests := <-E.AssignHallRequestsMapCh
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
		fmt.Println("assignedHallRequests -- ", assignedHallRequests)
	}
}

// Alias: Server()
func TCPReadElevatorStates(conn net.Conn, Backup ElevatorSystemChannels) {
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
			Backup.DisconnectedElevatorCh <- conn.LocalAddr().String() // Question: Should this be LocalAddr() or RemoteAddr() or both?
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
		case TypeActiveElevator:
			var msg MsgActiveElevator
			if err := json.Unmarshal(buf[:n], &msg); err != nil {
				log.Fatal(err)
			}
			fmt.Printf("Received ActiveElevator object: %+v\n", msg)
			Backup.StateUpdateCh <- msg.Content

		case TypeButtonEvent:
			var msg MsgButtonEvent
			if err := json.Unmarshal(buf[:n], &msg); err != nil {
				log.Fatal(err)
			}
			fmt.Printf("Received ButtonEvent object: %+v\n", msg)
			Backup.HallOrderCompleteCh <- msg.Content
		case TypeString:
			var msg MsgString
			if err := json.Unmarshal(buf[:n], &msg); err != nil {
				log.Fatal(err)
			}
			fmt.Printf("Received string object: %+v\n", msg)
			Backup.DisconnectedElevatorCh <- msg.Content

		default:
			fmt.Println("Unknown message type")
		}
	}
}

func TCPDialBackup(address string, port string) net.Conn {
	fmt.Println("TCPDialBackup() - Connecting by TCP to the address: ", address+port)

	conn, err := net.Dial("tcp", address+port)
	if err != nil {
		fmt.Println("Error in TCPDialBackup() - Connection failed. Error: ", err)
		return nil
	}

	fmt.Println("TCPDialBackup() - Conection established to: ", conn.RemoteAddr())
	return conn
}

// Can be used for testing purposes for writing either a ActiveElevator or ButtonEvent to TCPReadElevatorStates
func StartClient(port string, msg Message) {
	conn, err := net.Dial("tcp", "localhost"+port)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	data, err := json.Marshal(msg)
	if err != nil {
		log.Fatal(err)
	}

	_, err = conn.Write(data)
	if err != nil {
		log.Fatal(err)
	}
}

/*
distribute all hall requests
needs to receive ack from each elevator sendt to.
probably need to give it the TCP conn array

func DistributeHallRequests(assignedHallReq) {
	//TODO: all
}


distribute all button lights assosiated with each hallreq at each local elevator
needs to receive ack from each elevator sendt to.
probably need to give it the TCP conn array.
will need ack here aswell as hall req button lights need to be syncronized across computers
func DistributeHallButtonLights(assignedHallReq) {
	//TODO: all
}
*/

func ConnectedToNetwork() bool {
	conn, err := net.Dial("udp", "8.8.8.8:53") // (8.8.8.8 is a Google DNS)
	if err != nil {
		return false
	}
	defer conn.Close()
	return true
}

func RestartOnReconnect() {
	prevWasConnected := ConnectedToNetwork()
	for {
		if ConnectedToNetwork() && !prevWasConnected {
			fmt.Println("restarting stuffs:")
			exec.Command("gnome-terminal", "--", "go", "run", "./main.go").Run()
			panic("No network connection. Terminating current run - restarting from restart.go")
		}
		if ConnectedToNetwork() {
			prevWasConnected = true
		} else {
			prevWasConnected = false
		}
		time.Sleep(1 * time.Second)
	}
}

func GetLocalIPv4() string {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)

	return localAddr.IP.String()
}

func StringPortToInt(port string) int {
	portWithoutColon := strings.TrimPrefix(port, ":")

	portInt, _ := strconv.Atoi(portWithoutColon)

	return portInt
}
