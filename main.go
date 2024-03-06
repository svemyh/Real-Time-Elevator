package main

import (
	"elevator/elevio"
	"elevator/fsm"
	"fmt"
)

func main() {
	elevio.Init("localhost:15657", elevio.N_Floors)

	fmt.Printf("Started!\n")

	device := elevio.ElevInputDevice{
		FloorSensorCh:   make(chan int),
		RequestButtonCh: make(chan elevio.ButtonEvent),
		StopButtonCh:    make(chan bool),
		ObstructionCh:   make(chan bool),
	}
	//fsm_terminate := make(chan, bool)

	go elevio.PollFloorSensor(device.FloorSensorCh)
	go elevio.PollButtons(device.RequestButtonCh)
	go elevio.PollStopButton(device.StopButtonCh)
	go elevio.PollObstructionSwitch(device.ObstructionCh)

	go fsm.FsmRun(device)

	/*

			hraExecutable := ""
			switch runtime.GOOS {
			case "linux":
				hraExecutable = "hall_request_assigner"
			case "windows":
				hraExecutable = "hall_request_assigner.exe"
			default:
				panic("OS not supported")
			}

			input := hall_request_assigner.HRAInput{
				HallRequests: [][2]bool{{false, false}, {true, false}, {false, false}, {false, true}},
				States: map[string]hall_request_assigner.HRAElevState{
					"one": hall_request_assigner.HRAElevState{
						Behavior:    "moving",
						Floor:       2,
						Direction:   "up",
						CabRequests: []bool{false, false, false, true},
					},
					"two": hall_request_assigner.HRAElevState{
						Behavior:    "idle",yState := InitStateByBroadcastingNetworkAndWait()
		// If myState == PRIMARY {
		//		ActiveElevators[type array of structs Elevator] <- getAllElevatorStates()
		// 		sendOverNetworkToSecondary(ActiveElevators)
		//		for {
		//			ack <- network_recieved_ack
		//			time.sleep()
		//		}
		//      NewElevatorOrders = HallRequestAssigner(ActiveElevators[type array of structs Elevator])
		//		DistributeOrdersOverNetwork(NewElevatorOrders)
		// }
						Floor:       0,
						Direction:   "stop",
						CabRequests: []bool{false, false, false, false},
					},
					"three": hall_request_assigner.HRAElevState{
						Behavior:    "idle",
						Floor:       0,
						Direction:   "stop",
						CabRequests: []bool{false, false, false, false},
					}
					// TODO: recvret))
				//return
				}
			}

			output := new(map[string][][2]bool)
			err = json.Unmarshal(ret, &output)
			if err != nil {
				fmt.Println("json.Unmarshal error: ", err)
				return
			}

			fmt.Printf("output: \n")
			for k, v := range *output {
				fmt.Printf("%6v :  %+v\n", k, v)
			}
	*/
	select {}
	// 	InitLocalFSM()
	// 	if InitProcessPair() == "PRIMARY" {
	// 			go PrimaryRoutine()
	// 	}
	// 	else {
	// 		 establishConnectionWithPrimary()

	// 	}

	// 	func PrimaryRoutine(ActiveElevators) {

	// 			ActiveElevators[type array of structs Elevator] <- getAllACtiveElevatorStates()

	// 			Secondary = ActiveElevators[0]

	// 			sendPromotionMessageToSecondary(Secondary.MyAddress)

	// 			for {							// continue after acknowledgement
	// 				ack <- network_recieved_ack
	// 				time.sleep()
	// 			}

	// 			sendStatesOverNetworkToSecondary(ActiveElevators, Secondary.MyAddress)
	// 			for {
	// 				ack <- network_recieved_ack
	// 				time.sleep()
	// 			}

	// 	     NewElevatorOrders = HallRequestAssigner(ActiveElevators[type array of structs Elevator])
	// 			DistributeOrdersOverNetwork(NewElevatorOrders)
	// 	}

	// 	func SecondaryRoutine() {
	// 			initialize SecondarySinActiveElevators
	// 			go ListenToPrimary(chan messageCh, address)
	// 			case:
	// 				ActiveElevators<-messageCh:
	// 			SecondarySinActiveElevators = ActiveElevators

	// 	}

	// 	func RegularRoutine() {
	// 			go ListenForPromotion(msg chan<-string)
	// 			for {
	// 				select {
	// 			case: promotion <- msg
	// 				if promotion == "You are now secondary":
	// 					cancel ListenForPromotion()
	// 					go SecondaryRoutine()
	// 		}
	// 	}

	// 			// Do nothing
	// 	}

	// 	TODO:

	// 	func InitMyState()  {

	// 	}

	// 	func DistributeOrdersOverNetwork(NewElevatorOrders):
	// 			// TODO: Make this routine function in parallell instead of in series

	// 			for i in ActiveElevators:
	// 				sendOverNetwork(NewElevatorOrders[i])
	// 				for {
	// 					ack <- network_reciever
	// 					time.sleep()
	// 				}

	// 			for i in ActiveElevators:
	// 				sendOverNetwork(buttonlights)
	// 				for {
	// 					ack <- network_reciever
	// 					time.sleep()
	// 				}
	// 	}

	// 	chan networkReciever
	// 	func getAllElevatorStates(chan networkReciever)

	// 	case newEvent := <- networkReciever
	// 		 (MyAdress, elevio.elevator)
	// 	  ActiveElevators("MYIP") = elevio.elevator

	// }
}
