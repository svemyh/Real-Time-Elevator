## ReadMe for the Design Draft

#### Quick overview
The code revolves around running the local elevator FSM and the PrimaryBackup both as go routines.



#### Major aspects that are missing and need to be solved in design
1) No handling of a TCP conn disconnecting   ---> Could potentially add an LocalElevatorLost channel and have a goroutine that always checks what elevators are active (se kok)
2) No solution to potentially having 2 masters on network (should be easy to integrate tho) DONE
3) No handling of elevator loosing connection and reconnecting to network VEEEELDIG VIKTIG!!!!!! DONE
4) No idea how most of network function will work
5) Not started to add the conns to FSM module (should be easy) -will maybe include some mutexes assure that we update local state before sending it on the conn?

#### Potential places we can start to code step by step
	1) Sett opp en grunnlegende network modul (lag en server function, lag en client function, AmIPrimary(), broadcast på UDP ...) så vi kan ta i bruk i primary / secondary modul
	3) Hall request assigner (mulig alerede ferdig)
	4) Reasearch: Finn ut av TCP Ack, Finn ut av hvordan å sjekke om TCP conn dør og integrere i design hva vi gjør og hvordan vi håndterer en conn dør
	5) Start å sette opp en grunnlegende Primary / Backup dynamic ---> Alt utenom handlePrimaryTasks() (men må ta med lesing av conns fra handlePrimaryTasks())
		1) Finn ut av å lage en primary / backup. Få til å init en Primary og en Backup i RunPrimaryBackup()
		2) Få primary til å sende noe til backup
		3) Få til backup takeover til primary
