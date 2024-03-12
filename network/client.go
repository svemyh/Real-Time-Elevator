package network

import (
	"fmt"
	"net"
)

/*
import (
	"context"
	"fmt"
	"time"
)

type Secondary struct {
	Ip       string
	lastSeen time.Time
}

func SecondaryRoutine() {
	fmt.Println("Im a Secondary, doing Secondary things")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	//TODO: select to either receive or sent at any time
	Receiver(ctx, DETECTION_PORT)
}


func BecomeBackup() {

}
*/
// will simply be a net.Listen("TCP", "TCP_BACKUP_PORT"). This blocks code until a connection is established
func TCPListenForBackupPromotion(port string) (net.Conn, error) {
	fmt.Println(" - Executing TCPListenForBackupPromotion()")

	ls, err := net.Listen("tcp", port)
	if err != nil {
		fmt.Println("TCPListenForBackupPromotion - The connection failed. Error:", err)
		return nil, err
	}
	defer ls.Close()

	fmt.Println("TCPListenForBackupPromotion -  listening for new backup connections to port:", port)
	for {
		conn, err := ls.Accept()
		if err != nil {
			fmt.Println("Error: ", err)
			continue
		}
		return conn, nil
	}
}
