package network

import (
	"context"
	"fmt"
	"log"
	"net"
	"os/exec"
	"time"
)

//receiverChan := make(chan string)
//go network.Reciever(receiverChan, "localhost:20013")

func Receiver(ctx context.Context, receiver chan<- string, addressString string) {
	fmt.Println("Breakpoint: 0")
	addr, err := net.ResolveUDPAddr("udp", addressString) //addressString to actual address(server/)
	if err != nil {
		fmt.Println("Error resolving UDP address:", err)
		return
	}

	fmt.Println("Breakpoint: 1")
	// TODO: recvSock = new Socket(udp). Bind address we want to use to the socket
	recvSock, err := net.ListenUDP("udp", addr)
	if err != nil {
		fmt.Println("Error listening:", err)
		return
	}
	defer recvSock.Close() // Close recvSock AFTER surrounding main function completes

	buffer := make([]byte, 1024) // a buffer where the received network data is stored byte[1024] buffer
	fmt.Println("Breakpoint: 2")

	go func() {
		<-ctx.Done()
		recvSock.Close() // Close the socket when context is cancelled
	}()

	for {

		buffer = make([]byte, 1024)
		fmt.Println("Breakpoint: 3")

		numBytesReceived, fromWho, err := recvSock.ReadFromUDP(buffer)
		if err != nil {
			fmt.Println("Error readFromUDP:", err)
			return
		}
		message := string(buffer[:numBytesReceived])
		fmt.Println("Breakpoint: 4")

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
	laddr, err := net.ResolveUDPAddr("udp", "localhost:20015") // localIP
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

func InitProcessPair() string { // this should only return if master or slave, but I(Anton) am testing things using this function
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
		return currentRole
	// return 0. set local(?) elevatorsystem to Slave

	case <-time.After(3 * time.Second): // finn en måte å bruke setReadDeadline på. Resten funker
		if currentRole == "Slave" {
			log.Println("No message received, becoming master...")
			cancel() // should be redundant, but for some awkward reason it's not
			currentRole = "PRIMARY"
			return currentRole
			/*
				for {
					Sender("localhost:20017")
					log.Println(currentRole)
					time.Sleep(2 * time.Second)
					// master do master things
				}
			*/
		}
	}
	return currentRole
}

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
			fmt.Println("network yes")
		} else {
			prevWasConnected = false
			fmt.Println("network no")
		}
		time.Sleep(1 * time.Second)
	}
}
