package kafkacluster

import (
	"fmt"
	"net"
)

// CheckZookeeperIsReady returns True if zookeeper is ready or False and error
func CheckZookeeperIsReady(zookeeperHost string, zookeeperPort int32) (bool, error) {

	// connect to this socket
	zooStr := fmt.Sprintf("%s:%d", zookeeperHost, zookeeperPort)
	conn, err := net.Dial("tcp", zooStr)
	if err != nil {
		return false, err
	}
	defer conn.Close()

	fmt.Fprintf(conn, "ruok\n")

	// listen for reply
	var buf = make([]byte, 10)
	n, err := conn.Read(buf)
	if err != nil {
		return false, err
	}

	if string(buf[:n]) == "imok" {
		return true, nil
	}

	return false, nil
}
