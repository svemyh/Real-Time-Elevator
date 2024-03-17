package primary_backup

import (
	"net"
	"strings"
)

var localIP string

func LocalIP() (string, error) {
	if localIP == "" {
		conn, err := net.DialTCP("tcp4", nil, &net.TCPAddr{IP: []byte{8, 8, 8, 8}, Port: 53})
		if err != nil {
			return "", err
		}
		defer conn.Close()
		localIP = strings.Split(conn.LocalAddr().String(), ":")[0]
	}
	return localIP, nil
}

func RemoteIP(conn net.Conn) string {
	remoteAddr := conn.RemoteAddr().String()
	ip := strings.Split(remoteAddr, ":")[0]
	return ip
}
