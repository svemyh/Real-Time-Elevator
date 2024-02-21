package network

import (
	"fmt"
	"net"
)

//receiverChan := make(chan string)
//go network.Reciever(receiverChan, "localhost:20013")

func Reciever(receiver chan<- string, addressString string) {
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
	defer recvSock.Close() // Close recvSock AFTER surrounding main function completes

	buffer := make([]byte, 1024) // a buffer where the received network data is stored byte[1024] buffer
	fmt.Println("Breakpoints")

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
			receiver <- message
		} else {
			fmt.Println("rand message is: ", message)
			receiver <- message
		}
	}
}
