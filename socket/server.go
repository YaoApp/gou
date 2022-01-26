package socket

import (
	"fmt"
	"log"
	"net"
	"os"
)

// Socket server (alpha)

// Start 启动服务
func Start(proto string, host string, port string, handle func([]byte) []byte) {
	if proto == "tcp" {
		tcpStart(host, port, handle)
	}
}

func tcpStart(host string, port string, handle func([]byte) []byte) {
	listen, err := net.Listen("tcp", fmt.Sprintf("%s:%s", host, port))
	if err != nil {
		fmt.Println("Socket Start error:", err)
		os.Exit(1)
	}
	defer listen.Close()
	fmt.Printf("Listening ON: %s:%s\n", host, port)

	for {
		conn, err := listen.Accept()
		if err != nil {
			fmt.Println("Error Accepting: ", err)
			os.Exit(1)
		}
		//logs an incoming message
		fmt.Printf("Received message %s -> %s \n", conn.RemoteAddr(), conn.LocalAddr())
		// Handle connections in a new goroutine.
		go handleRequest(conn, handle)
	}
}

/**
100000012021124947278772  152d02f2967138c87bb4  抗金属 纽扣
100000012021129634062032  152d02f2967250231ed0  抗金属 纽扣 小
100000012021123642243533  152d02f29670eaff39cd  产品设计
100000012021128798321101  152d02f296721e52b9cd  易经
100000012021129634062011  152d02f2967250231ebb  增长黑客
15 2d 02 f2 96 72 1e 52 b9 cd  易经
*/
func handleRequest(conn net.Conn, handle func([]byte) []byte) {
	clientAddr := conn.RemoteAddr().String()
	defer conn.Close()
	log.Println("Connection success. Client address: ", clientAddr)
	for {
		buffer := make([]byte, 1024)
		recvLen, err := conn.Read(buffer)
		if err != nil {
			log.Println("Read error: ", err, clientAddr)
			return
		}

		res := handle(buffer[:recvLen])
		if res != nil {
			if _, err := conn.Write(res); err != nil {
				log.Println("Write error: ", err, clientAddr)
				return
			}
		}
	}
}
