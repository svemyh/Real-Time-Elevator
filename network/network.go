package network

import (
	"context"
	"fmt"
	"log"
	"net"
	"time"
)

var detectionPort string = ":20017"

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

func InitNetwork(ctx context.Context) {
	isPrimary := AmIPrimary(detectionPort)
	if isPrimary {
		go UDPBroadCastPrimaryRole(ctx, detectionPort)
		log.Println("Operating as primary...")
		go PrimaryRoutine()
	} else {
		log.Println("Operating as client...")
		go SecondaryRoutine()
	}
}

//receiverChan := make(chan string)
//go network.Reciever(receiverChan, "localhost:20013")

/*


func TCPReadElevatorStates(conn, StateUpdateCh, OrderCompleteCh) {
	//TODO:Read the states and store in a buffer
	//TODO: Check if the read data was due to local elevator reaching a floor and clearing a request (send cleared request on OrderCompleteCh)
	//TODO:send the updated states on stateUpdateCh so that it can be read in HandlePrimaryTasks(StateUpdateCh)
}

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
