InitLocalFSM()


if InitProcessPair() == "PRIMARY" {
		go PrimaryRoutine()
}
else {
	go RegularRoutine()
	establishConnectionWithPrimary() // TCP
	
}


func getAllACtiveElevatorStates()  elevator.Elevator {
	//1)receive if any elevator on network changes states locally (whole elevator struct)
	//2)receive if 1 elevator dies (includes network disconnect and any local error such as power loss to computer or elevator motors) and remove from activeElevatorArray
	//3)recieve that a new elevator is on network
	//2 and 3 will be noticed by step 1 as a new tcp message is sendt or a established tcp conn is removed.

	go ListenForEventsOnOtherElevators(chan eventCh string)
	case event<-eventCh:
		ActiveElevators = UpdateActiveElevators(ActiveElevators, event)



	return ActiveElevators // type: array of structs Elevator]
}

func PrimaryRoutine(ActiveElevators) {
	go getAllACtiveElevatorStates()
	select {	
		ActiveElevators <- 
	}
	Secondary = ActiveElevators[0] //should ideally happen only once and happen again if the secondary assigned elevator dies
	//if we decide to change to some logic to decide new secondary, this should be time based so a new secondary is not chosen constantly
	//if ActiveElevators[0] remais unchanged as long as the elvator still is alive then we could do
	//before loop
	//Secondary = ActiveElevators[0]
	//in loop:
	//if ActiveElevators[0] != Secondary{ Secondary = ActiveElevators[0] } ----> will only run if the asssigned secondary dies
	
	sendPromotionMessageToSecondary(Secondary.MyAddress)
	for {							// continue after acknowledgement
		ack <- network_recieved_ack
		time.sleep()
	}

	sendStatesOverNetworkToSecondary(ActiveElevators, Secondary.MyAddress)
	for {
		ack <- network_recieved_ack
		time.sleep()
	}

	NewElevatorOrders = HallRequestAssigner(ActiveElevators[type array of structs Elevator])
	DistributeOrdersOverNetwork(NewElevatorOrders)
}


func SecondaryRoutine() {
		// TODO: Husk å implementere ACK til Primary.

		var SecondarySinActiveElevators 	 //store ActiveElevators recieved from Primary
		go ListenToPrimary(chan messageCh, address)
		for {
			select {
				case ActiveElevators<-messageCh:
					SecondarySinActiveElevators = ActiveElevators
					Conn.setReadDeadline(time.now().Add(3*time.Second))
				case <- Conn.deadlineReached:
					go PrimaryRoutine(SecondarySinActiveElevators)
			}
		}
}


func RegularRoutine() {
		go ListenForPromotion(msg chan<-string) 
		for {
			select {
			case promotion <- msg:
				if promotion == "You are now secondary"{
					cancel ListenForPromotion()
					go SecondaryRoutine()
				}
			default:
				// 
			}
		}	
}



func DistributeOrdersOverNetwork(NewElevatorOrders) {
		// TODO: Make this routine function in parallell instead of in series

		for i in ActiveElevators:
			sendOverNetwork(NewElevatorOrders[i])
			for {
				ack <- network_reciever
				time.sleep()
			}

		for i in ActiveElevators:
			sendOverNetwork(buttonlights)
			for {
				ack <- network_reciever
				time.sleep()
			}
}




chan networkReciever
func getAllElevatorStates(chan networkReciever)

case newEvent := <- networkReciever
		(MyAddress, elevio.elevator)
	ActiveElevators("MYIP") = elevio.elevator

