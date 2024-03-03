//DRAFT FILE FOR NETWORK

func TCPReadElevatorStates(conn, StateUpdateCh) {
	//TODO:Read the states and store in a buffer
	//TODO:send the updated states on stateUpdateCh so that it can be read in HandlePrimaryTasks(StateUpdateCh)
}

func TCPListenForNewPrimary() {
	//listen for new primary on tcp port and accept
}

func TCPListenForNewElevators(){
	//listen for new elevators on TCP port
	//when connection established run the go routine to start reading data from the conn
	go run TCPReadElevatorStates(stateUpdateCh)
}


func UDPBroadCastPrimaryRole() {
	//TODO: broadcast that you are primary 
}

/*
distribute all hall requests
needs to receive ack from each elevator sendt to.
probably need to give it the TCP conn array
*/
func DistributeHallRequests(assignedHallReq) {
	//TODO: all
}	

/*
distribute all button lights assosiated with each hallreq at each local elevator
needs to receive ack from each elevator sendt to.
probably need to give it the TCP conn array.
will need ack here aswell as hall req button lights need to be syncronized across computers
*/
func DistributeHallButtonLights(assignedHallReq) {
	//TODO: all
}