package network

import (
	"context"
	"fmt"
	"log"
	"net"
	"time"
)

//receiverChan := make(chan string)
//go network.Reciever(receiverChan, "localhost:20013")

func Receiver(ctx context.Context, receiver chan<- string, addressString string) {
	fmt.Println("Breakpoints")
	addr, err := net.ResolveUDPAddr("udp", addressString) //addressString to actual address(server/)
	if err != nil {
		fmt.Println("Error resolving UDP address:", err)
		return
	}

	fmt.Println("Breakpoints")
	// TODO: recvSock = new Socket(udp). Bind address we want to use to the socket
	recvSock, err := net.ListenUDP("udp", addr)
	if err != nil {
		fmt.Println("Error listening:", err)
		return
	}
	//defer recvSock.Close() // Close recvSock AFTER surrounding main function completes

	buffer := make([]byte, 1024) // a buffer where the received network data is stored byte[1024] buffer
	fmt.Println("Breakpoints")

	go func() {
		<-ctx.Done()
		recvSock.Close() // Close the socket when context is cancelled
	}()

	for {

		buffer = make([]byte, 1024)

		numBytesReceived, fromWho, err := recvSock.ReadFromUDP(buffer)
		if err != nil {
			fmt.Println("Error readFromUDP:", err)
			return
		}
		fmt.Println("Breakpoints")
		message := string(buffer[:numBytesReceived])

		localIP, err := net.ResolveUDPAddr("udp", addressString) // localIP
		if err != nil {
			fmt.Println("Error resolving UDP address:", err)
			return
		}

		if string(fromWho.IP) != string(localIP.IP) {
			fmt.Println(message)
			//fmt.PrintIn("Filtered out: ", string(buffer[0:numBytesReceived]))
			//receiver <- messageInitStateByBroadcastingNetworkAndWait()
		} else {
			fmt.Println("rand message is: ", message)
			receiver <- message
		}
	}
}

func Sender(port string) {
	laddr, err := net.ResolveUDPAddr("udp", "localhost:20017") // localIP
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
	sockConn, _ := net.DialUDP("udp", laddr, raddr)

	defer sockConn.Close()

	//Create an empty buffer to to filled
	buffer := make([]byte, 1024)
	//oob := make([]byte, 1024)

	n := copy(buffer, []byte("Hello World!"))

	fmt.Printf("copied %d bytes to the buffer: %s\n", n, buffer[:n])

	n, err = sockConn.Write(buffer)

	if err != nil {
		fmt.Println("Error resolving WriteMsgUDP", err)
		return
	}
}

func InitStateByBroadcastingNetworkAndWait() {
	var (
		currentRole = "Slave" // Start as receiver
	)

	receiverChan := make(chan string)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go Receiver(ctx, receiverChan, "localhost:20017")

	select {
	case msg := <-receiverChan:
		log.Println("Received message:", msg)
	case <-time.After(3 * time.Second):
		if currentRole == "Slave" {
			log.Println("No message received, becoming master...")
			cancel() // should be redundant, but for some awkward reason it's not
			currentRole = "Master"
			for {
				go Sender("localhost:20017")
				time.Sleep(5 * time.Second)
				// master do master things
			}
		}
	}
}
