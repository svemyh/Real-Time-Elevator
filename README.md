Elevator Project
================

### Overview

This project implements a distributed control system for managing `n` elevators in real-time across `m` floors. The system dynamically assigns floor orders to available elevators, optimizes elevator movements, and handles elevator malfunctions (e.g., stuck elevators and disconnected elevators) to ensure efficient operation. 

### Our solution
We wrote our solution in Google Go. To acheive the desired fault tolerance we have multiple computers communicating with each other. We then chose to implement a primary/backup structure where all elevators operate as a "Local Elevators" and 2 of the elevators will also be promoted to a Primary role and Backup role respectivly. The primary elevator has extra responsibility to:

   - Collect the state of every elevator
   - Gather all orders and order completions from all elevators
   - Calculate the resulting order assignments
   - Send orders and assignments to all elevators

The Backup is responsible for keeping an exact copy of al the information that the primary has. If the primary loses internet connection or stops responding for other reasons the backup will be promoted to a primary. It will then promote another backup to replace it (if another elevator is available).

The communication between elevators is done with a combination of TCP and UDP. There is a TCP connection between the primary and every availible local elevator in order to ensure reliable network communication between them. We also have a seperate TCP connection between the Primary and Backup. Our solution also uses UDP broadcasts. The primary will broadcast its role repeatedly to all elevators on the network. If an elevator enters the network it will check if there is a primary, if not it will promote itself to a primary. Otherwise it will remain as a simple local elevator. All elevators are also responsible for broadcasting that they are still connected to the network through UDP. The Primary implements a simple counter based in this to reliably notice when an elevator has been disconnected.
All signals from the primary to local elevators regarding which hall request button lights should be active are also broadcasted by UDP. 

### Key features

- **Dynamic hall Request assignment:** Efficiently distributes hall requests among active elevators based on their current state and location.
- **Real-time monitoring:** Monitors the operational status of each elevator, adjusting assignments as elevators become available or stuck.
- **Fault tolerance:** Detects and responds to elevator malfunctions, removing stuck elevators from the pool of active units until they are operational again.

### System Components

1. **Elevator module:** Simulates elevator operations, including moving between floors, opening/closing doors, and executing floor requests.
2. **Hall request assigner:** Allocates hall requests to elevators based on their current state and proximity to requested floors.
3. **Finite state machine** Implements a state machine to control a local elevator based on their current state
4  **Elevio** Facilitates communication between input / output between the elevator software and elevator hardware such as detecting button presses etc. 
5  **Timer** implements simple timer functions such as starting a timer, stoping a timer and checking for a time-out
6  **primary_Backup:** Facilitates implements all logic needed to implement the primary and backup role architecture mentioned above. Contains the following sub-modules
7  **Conn** Includes a function to enable UDP broadvasting on a local network
   1) **Primary** Manages the overall system state, including tracking active elevators and their statuses, and runs the hall request assigner logic.
   2) **Backup** Takes over Primary Controller should the primary elevator fail, constantly keeping track of active elevators and their statuses for quick recovery
   3) **Network** Contains all functions that are exclusivly network related and independent on an elevator system

### Running the system

# Testing on a model elevator
1. Give permission to `hall_request_assigner` and `elevatorserver`elevatorserver using `chmod +x filename`
2. Launch `elevatorserver` in a terminal
3. Launch `go run main.go` in a terminal

# If you want to run from a simulator
1. Give permission to SimElevatorServer by using `chmod +x SimElevatorServer`
2. Launch `./SimElevatorServer` in a terminal
3. Launch `go run main.go` in a terminal
4. The default port of the simulation is set to 15657. If you want to test on a specific port, enter `./SimElevatorServer --port <your-port>`, and replace `<your-port>` with your desired port number


### Prerequisites

- Go (version 1.14 or later) installed on your system.
- Access to a Linux operating system for deployment.Elevator Project

  ## Contributors
<table>
  <tr>
    <td align="center">
        <a href="https://github.com/MikaelShahly">
            <img src="https://github.com/MikaelShahly.png?size=100" width="100px;" alt="Mikael Shahly"/><br />
            <sub><b>Mikael Shahly</b></sub>
        </a>
    </td>
    <td align="center">
        <a href="https://github.com/AntTra">
            <img src="https://github.com/AntTra.png?size=100" width="100px;" alt="Anton Tran"/><br />
            <sub><b>Anton Tran</b></sub>
        </a>
    </td>
    <td align="center">
        <a href="https://github.com/svemyh">
            <img src="https://github.com/svemyh.png?size=100" width="100px;" alt="Sveinung Myhre"/><br />
            <sub><b>Sveinung Myhre</b></sub>
        </a>
    </td>
  </tr>
</table>
