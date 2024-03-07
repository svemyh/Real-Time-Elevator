package network

import (
	"context"
	"fmt"
	"time"
)

type Backup struct {
	Ip       string
	lastSeen time.Time
}

func BackupRoutine() {
	fmt.Println("Im a backup, doing backup things")
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
