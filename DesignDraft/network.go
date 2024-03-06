//DRAFT FILE FOR NETWORK

/*
This will run as a goroutine that handles anything that primary connects to OR that connects to a primary.
So in the primary side, this will run for any connection that is connected to primary.
Is responsible to read the data sendt from each "local elevator" connected to the primary
The "local elevator" will then have to write in FSM function when a button press etc etc happens
return: nothing
*/
func TCPReadElevatorStates(conn, StateUpdateCh, OrderCompleteCh) {
	//TODO:Read the states and store in a buffer
	//TODO: Check if the read data was due to local elevator reaching a floor and clearing a request (send cleared request on OrderCompleteCh)
	//TODO:send the updated states on stateUpdateCh so that it can be read in HandlePrimaryTasks(StateUpdateCh)
}

/*
Listens for the case that a backup has become a primary and tries to dial every previous elevator the previous primary
had contact with.
Every elevator will be listening here
Return: nothing
*/
func TCPListenForNewPrimary() {
	//TODO: listen for new primary on tcp port and accept
}

/*
The primary will run this function to listen if a new elevator enters network and wants to establish a
TCP connection with it.
return: NOTHING
*/
func TCPListenForNewElevators(){
	//listen for new elevators on TCP port
	//when connection established run the go routine TCPReadElevatorStates to start reading data from the conn
	go run TCPReadElevatorStates(stateUpdateCh)
}


func UDPBroadCastPrimaryRole() {
	//TODO: broadcast that you are primary 
	
}

/*
distribute all hall requests
needs to receive ack from each elevator sendt to.
probably need to give it the TCP conn array
Return: nothing
*/
func DistributeHallRequests(assignedHallReq) {
	//TODO: all
}	

/*
distribute all button lights assosiated with each hallreq to each local elevator
needs to receive ack from each elevator sendt to.
probably need to give it the TCP conn array.
will need ack here aswell as hall req button lights need to be syncronized across computers
return: nothing
*/
func DistributeHallButtonLights(assignedHallReq) {
	//TODO: all
}