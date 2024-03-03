//DRAFT FILE FOR BACKUP

/*
establish connection with all elevators primary previously had connection with 
	-Does it matter that primary now acts as client and not server? -> no as long as a conn object is made it matters not
	-In primary routine we still need to listen to new TCP connections
	-Requires that the local elevators are listening for a client
Run primaryRoutine now that we have re-established all connections with prev elevators
*/
BecomePrimary(elevatorStates []Elevator) {
	//TODO: create arrayOfConn
	//establish TCP connection with all elevators last primary had connections with as a client
	//has to get a TCP conn object with all new elevators to be able to interact in PrimaryRoutine
	//Needs to run a goroutine TCPReadElevatorStates when connection established
	//Note, the elevator list will contain your own IP. DO NOT CONNECT TO IT, as PrimaryRoutine will make this connection 
	//(OR HANDLE HERE IF DESIRED)
	PrimaryRoutine(arrayOfConn)
}

BackupRoutine() {
	//TODO: Moniter that primary connection is alive
	//TODO: read states sendt through primary connection
	//TODO: if backup unresponsive --> BecomePrimary()
}

