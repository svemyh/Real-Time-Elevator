package network

import (
	"context"
	"elevator/elevio"
	"elevator/hall_request_assigner"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os/exec"
	"time"
)

var DETECTION_PORT string = ":10002"
var TCP_LISTEN_PORT string = ":10001"
var TCP_BACKUP_PORT string = ":15000"

type MessageType string

const (
	TypeActiveElevator MessageType = "ActiveElevator"
	TypeButtonEvent    MessageType = "ButtonEvent"
)

type Message interface{}

type MsgActiveElevator struct {
	Type    MessageType                          `json:"type"`
	Content hall_request_assigner.ActiveElevator "json:content"
}

type MsgButtonEvent struct {
	Type    MessageType        `json:"type"`
	Content elevio.ButtonEvent "json:content"
}

/*
type RXChannels struct {
	StateUpdateCh       chan types.ElevState      `addr:"stateupdatech"`
	RegisterOrderCh     chan types.OrderEvent     `addr:"registerorderch"`
	OrdersFromMasterCh  chan types.GlobalOrderMap `addr:"ordersfrommasterch"`
	OrderCopyRequestCh  chan bool                 `addr:"ordercopyrequestch"`
	OrderCopyResponseCh chan types.GlobalOrderMap `addr:"ordercopyresponsech"`
}
*/

func InitReceiver(ctx context.Context, receiver chan<- string, addressString string) string {
	addr, err := net.ResolveUDPAddr("udp", addressString) //addressString to actual address(server/)
	if err != nil {
		fmt.Println("Error resolving UDP address:", err)
	}

	// TODO: recvSock = new Socket(udp). Bind address we want to use to the socket
	recvSock, err := net.ListenUDP("udp", addr)
	if err != nil {
		fmt.Println("Error listening:", err)
	}
	defer recvSock.Close() // Close recvSock AFTER surrounding main function completes

	//buffer := make([]byte, 1024) // a buffer where the received network data is stored byte[1024] buffer

	for {
		select {
		case <-ctx.Done():
			break
		default:
			recvSock.SetReadDeadline(time.Now().Add(3 * time.Second))
			buffer := make([]byte, 1024)

			numBytesReceived, fromWho, err := recvSock.ReadFromUDP(buffer)
			if err != nil {
				fmt.Println("Error readFromUDP:", err)
				break
			}
			message := string(buffer[:numBytesReceived])

			localIP, err := net.ResolveUDPAddr("udp", addressString) // localIP
			if err != nil {
				fmt.Println("Error resolving UDP address:", err)
				break
			}

			if string(fromWho.IP) != string(localIP.IP) {
				fmt.Printf("Received: %s\n", message)
				//fmt.PrintIn("Filtered out: ", string(buffer[0:numBytesReceived]))
				//receiver <- messageInitStateByBroadcastingNetworkAndWait()
				return message
			}
		}
	}
}

func Receiver(ctx context.Context, TCPPort string) {
	ls, err := net.Listen("tcp", TCPPort)
	if err != nil {
		fmt.Println("The connection failed. Error:", err)
		return
	}
	defer ls.Close()

	fmt.Println("Connected to port:", TCPPort)
	for {
		conn, err := ls.Accept()
		if err != nil {
			fmt.Println("Error: ", err)
			continue
		}

		go handleConnection(conn)
		time.Sleep(1 * time.Second)
	}
}

func handleConnection(conn net.Conn) {
	defer conn.Close()

	//Accept
	fmt.Printf("Accepted connection from %s\n", conn.LocalAddr())

	//Send
	msg := fmt.Sprintf("Connect to: %s\n", conn.LocalAddr())
	conn.Write([]byte(msg))
}

func Transmitter(TCPPort string) {
	conn, err := net.Dial("tcp", TCPPort)
	if err != nil {
		fmt.Println("The connection failed. Error: ", err)
		return
	} else {
		fmt.Printf("The connection was established to: %s \n", conn.RemoteAddr()) //wtf
	}

	conn.Write([]byte("From client!"))

	buffer := make([]byte, 1024)
	bytes, err := conn.Read(buffer)
	if err != nil {
		fmt.Println("Error resolving UDP address:", err)
		return
	}

	fmt.Println(string(buffer[:bytes]))

	defer conn.Close()
}

// Alias: RunPrimaryBackup()
func InitNetwork(ctx context.Context, FSMStateUpdateCh chan hall_request_assigner.ActiveElevator, FSMHallOrderCompleteCh chan elevio.ButtonEvent, StateUpdateCh chan hall_request_assigner.ActiveElevator, HallOrderCompleteCh chan elevio.ButtonEvent, DisconnectedElevatorCh chan string) {
	isPrimary, primaryAddress := AmIPrimary(DETECTION_PORT)
	if isPrimary {
		log.Println("Operating as primary...")
		go PrimaryRoutine(StateUpdateCh, HallOrderCompleteCh, DisconnectedElevatorCh)
		time.Sleep(1500 * time.Millisecond)
		TCPDialPrimary("localhost"+TCP_LISTEN_PORT, FSMStateUpdateCh, FSMHallOrderCompleteCh)
	} else {
		log.Println("Operating as client...")
		go TCPDialPrimary(primaryAddress, FSMStateUpdateCh, FSMHallOrderCompleteCh)
		go TCPListenForNewPrimary(TCP_LISTEN_PORT, FSMStateUpdateCh, FSMHallOrderCompleteCh)
		conn, _ := TCPListenForBackupPromotion(TCP_BACKUP_PORT) //will simply be a net.Listen("TCP", "primaryAdder"). This blocks code until a connection is established
		BackupRoutine(conn, primaryAddress)
	}
}

// Checks the event that a backup has become a new primary and wants to establish connection. This go routine should be shut down at some point
func TCPListenForNewPrimary(TCPPort string, FSMStateUpdateCh chan hall_request_assigner.ActiveElevator, FSMHallOrderCompleteCh chan elevio.ButtonEvent) {
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

	fmt.Println("Primary is listening for new connections to port:", TCPPort)
	for {
		conn, err := ls.Accept()
		if err != nil {
			fmt.Println("Error: ", err)
			continue
		}
		go sendLocalStatesToPrimaryLoop(conn, FSMStateUpdateCh, FSMHallOrderCompleteCh) // This will terminate whenever the connection/conn is closed - i.e. conn.Write() throws an error.
	}
}

// will simply be a net.Listen("TCP", "TCP_BACKUP_PORT"). This blocks code until a connection is established
func TCPListenForBackupPromotion(port string) (net.Conn, error) {
	fmt.Println(" - Executing TCPListenForBackupPromotion()")

	ls, err := net.Listen("tcp", port)
	if err != nil {
		fmt.Println("The connection failed. Error:", err)
		return nil, err
	}
	defer ls.Close()

	fmt.Println("Primary is listening for new connections to port:", port)
	for {
		conn, err := ls.Accept()
		if err != nil {
			fmt.Println("Error: ", err)
			continue
		}
		return conn, nil
	}
}

func SecondaryRoutine(conn net.Conn) {
	fmt.Println("I'm a Secondary, doing Secondary things")
}

func TCPDialPrimary(PrimaryAddress string, FSMStateUpdateCh chan hall_request_assigner.ActiveElevator, FSMHallOrderCompleteCh chan elevio.ButtonEvent) {
	fmt.Println("Connecting by TCP to the address: ", PrimaryAddress)

	conn, err := net.Dial("tcp", PrimaryAddress)
	if err != nil {
		fmt.Println("Connection failed. Error: ", err)
		return
	}

	fmt.Println("Conection established to: ", conn.RemoteAddr())
	defer conn.Close()

	sendLocalStatesToPrimaryLoop(conn, FSMStateUpdateCh, FSMHallOrderCompleteCh)
}

func sendLocalStatesToPrimaryLoop(conn net.Conn, FSMStateUpdateCh chan hall_request_assigner.ActiveElevator, FSMHallOrderCompleteCh chan elevio.ButtonEvent) {
	fmt.Println("Conection established to: ", conn.RemoteAddr())
	defer conn.Close()

	for {
		select {
		case stateUpdate := <-FSMStateUpdateCh:
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

		case hallOrderComplete := <-FSMHallOrderCompleteCh:
			my_ButtonEventMsg := MsgButtonEvent{
				Type:    TypeButtonEvent,
				Content: hallOrderComplete,
			}

			data, err := json.Marshal(my_ButtonEventMsg)
			if err != nil {
				fmt.Println("Error sending to Primary: ", err)
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

// Alias: Server()
func TCPReadElevatorStates(conn net.Conn, StateUpdateCh chan hall_request_assigner.ActiveElevator, HallOrderCompleteCh chan elevio.ButtonEvent, DisconnectedElevatorCh chan string) {
	//TODO:Read the states and store in a buffer
	//TODO: Check if the read data was due to local elevator reaching a floor and clearing a request (send cleared request on OrderCompleteCh)
	//TODO:send the updated states on stateUpdateCh so that it can be read in HandlePrimaryTasks(StateUpdateCh)
	// Can be added/expanded: LocalErrorDetectedCh or similar
	// type StateUpdateCh = IP + elevatorStates
	// type HallOrderCopleteCh = floor number (of cab call completed)

	fmt.Printf("*New connection accepted from adress: %s\n", conn.LocalAddr())

	defer conn.Close()

	for {
		fmt.Printf("STILL IN READING LOOP PRIMARY SIDE")
		// Create buffer and read data into the buffer using conn.Read()
		var buf [1024]byte
		n, err := conn.Read(buf[:])
		if err != nil {
			// Error means TCP-conn has broken -> Need to feed this signal to drop the conn's respective ActiveElevator from Primary's ActiveElevators. It is now considered inactive.
			DisconnectedElevatorCh <- conn.LocalAddr().String()
			log.Fatal(err)
		}

		// Decoding said data into a json-style object
		var genericMsg map[string]interface{}
		if err := json.Unmarshal(buf[:n], &genericMsg); err != nil {
			fmt.Println("genericMsg: ", genericMsg)
			fmt.Println("Error unmarshaling generic message: ", err)
			log.Fatal(err)
		}
		fmt.Println("genericMsg NORMAL: ", genericMsg)
		// Based on MessageType (which is an element of each struct sent over connection) determine how its corresponding data should be decoded.
		switch MessageType(genericMsg["type"].(string)) {
		case TypeActiveElevator:
			var msg MsgActiveElevator
			if err := json.Unmarshal(buf[:n], &msg); err != nil {
				log.Fatal(err)
			}
			fmt.Printf("Received ActiveElevator object: %+v\n", msg)
			StateUpdateCh <- msg.Content

		case TypeButtonEvent:
			var msg MsgButtonEvent
			if err := json.Unmarshal(buf[:n], &msg); err != nil {
				log.Fatal(err)
			}
			fmt.Printf("Received ButtonEvent object: %+v\n", msg)
			HallOrderCompleteCh <- msg.Content

		default:
			fmt.Println("Unknown message type")
		}
	}
}

func TCPDialBackup(address string, port string) net.Conn {
	fmt.Println("Connecting by TCP to the address: ", address)

	conn, err := net.Dial("tcp", address)
	if err != nil {
		fmt.Println("Connection failed. Error: ", err)
		return nil
	}

	fmt.Println("Conection established to: ", conn.RemoteAddr())
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
func TCPListenForNewPrimary() {
	//listen for new primary on tcp port and accept
}

func TCPListenForNewElevators(port string, listenerconnection, receiverchannels){
	//listen for new elevators on TCP port
	//when connection established run the go routine TCPReadElevatorStates to start reading data from the conn
	go run TCPReadElevatorStates(stateUpdateCh)

	allClients := make(map[net.Conn]string)
	newConnections := make(chan net.Conn)
	deadConnections := make(chan net.Conn)
	messages := make(chan connectionMsg)

	go acceptConnections(connection, newConnections)

	for {
		select {
		case conn := <-newConnections:
			addr := conn.RemoteAddr().String()
			fmt.Printf("Accepted new client, %v\n", addr)
			allClients[conn] = addr
			go read(conn, messages, deadConnections)

		case conn := <-deadConnections:
			fmt.Printf("Client %v disconnected", allClients[conn])
			delete(allClients, conn)

		case message := <-messages:
			go decodeMsg(message, rxChannels)
		}
	}
}

func acceptConnections(server net.Listener, newConnections chan net.Conn) {
	for {
		conn, err := server.Accept()
		if err != nil {
			fmt.Println(err)
		}
		newConnections <- conn
	}
}
*/

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
		if (ConnectedToNetwork()) && (prevWasConnected == false) {
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
