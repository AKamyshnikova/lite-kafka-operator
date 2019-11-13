package kafkacluster

import (
	"bufio"
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

	fmt.Fprintf(conn, "ruok\n")
	// listen for reply
	var data []byte
	data = make([]byte, 4)
	_, err = bufio.NewReader(conn).Read(data)
	if err != nil {
		return false, err
	}

	if string(data) == "imok" {
		return true, nil
	}

	return false, nil
}
