package kafkacluster

import (
	"fmt"
	"net"
	"time"
)

// CheckZookeeperIsReady returns True if zookeeper is ready or False and error
func CheckZookeeperIsReady(zookeeperHost string, zookeeperPort int32) (bool, error) {

	// connect to this socket
	zooStr := fmt.Sprintf("%s:%d", zookeeperHost, zookeeperPort)
	dialer := net.Dialer{Timeout: time.Duration(time.Second * 2)}
	conn, err := dialer.Dial("tcp", zooStr)
	if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
		return false, nil
	}
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

func PollZookeeperCheck(zookeeperHost string, zookeeperPort int32) (bool, error) {
	tries := 10
	for i := 0; i < tries; i++ {
		ok, err := CheckZookeeperIsReady(zookeeperHost, zookeeperPort)
		if err != nil {
			return false, err
		}
		if ok {
			return true, nil
		}
	}
	return false, nil
}
