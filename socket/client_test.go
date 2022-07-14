package socket

import (
	"fmt"
	"net"
	"os"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/go-playground/assert/v2"
)

var done = make(chan uint)
var events = map[string]chan uint{
	"onConnected": make(chan uint),
	"onData":      make(chan uint),
	"onError":     make(chan uint),
	"onClosed":    make(chan uint),
}
var resps = map[string]interface{}{}

func TestClient(t *testing.T) {

	srv := server()
	go request(srv, done)
	addr := strings.Split(srv.Addr().String(), ":")
	host := addr[0]
	port := addr[1]

	client := NewClient(
		Option{
			Protocol:   "tcp",
			Host:       host,
			Port:       port,
			Timeout:    5 * time.Second,
			BufferSize: 16,
		},
		Handlers{
			Connected: onConnected,
			Error:     onError,
			Data:      onData,
			Closed:    onClosed,
		})

	go func() { client.Open() }()

	go func() {
		time.Sleep(2000 * time.Millisecond)
		done <- 1
	}()

	// onConnected
	if ok := <-events["onConnected"]; ok == 1 {
		assert.Equal(t, host, resps["onConnected"].(Option).Host)
		assert.Equal(t, port, resps["onConnected"].(Option).Port)
		resps["onConnected"] = nil
	}

	// onData
	if ok := <-events["onData"]; ok == 1 {
		assert.Equal(t, "48656c6c6f|5", resps["onData"]) // Hello|5
	}

	// Close connection Testing Auto-Reconnect
	client.Conn.Close()

	// onConnected
	if ok := <-events["onConnected"]; ok == 1 {
		assert.Equal(t, host, resps["onConnected"].(Option).Host)
		assert.Equal(t, port, resps["onConnected"].(Option).Port)
	}

	msg := <-done
	fmt.Println(host, port)
	syscall.Kill(syscall.Getpid(), syscall.SIGINT)

	// Manal Close
	if ok := <-events["onData"]; ok == 1 {
		fmt.Println("resps:", resps["onData"])
	}

	// close
	// go func() { done <- 1 }()
	msg = <-done
	fmt.Println(msg)
}

func onConnected(option Option) error {
	fmt.Println("onConnected:", option.Port)
	resps["onConnected"] = option
	events["onConnected"] <- 1
	return nil
}

func onData(data []byte, length int) ([]byte, error) {
	fmt.Printf("onData: %x|%d\n", data, length)
	resps["onData"] = fmt.Sprintf("%x|%d", data, length)
	events["onData"] <- 1
	return nil, nil
}

func onClosed(msg []byte, err error) []byte {
	resps["onClosed"] = fmt.Sprintf("%v|%v", msg, err)
	fmt.Println("onClosed: called")
	return []byte("Gone")
}

func onError(err error) {
	resps["onError"] = err
	fmt.Println("onError", err)
}

func server() net.Listener {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		fmt.Println("Error listening:", err.Error())
		os.Exit(1)
	}
	return l
}

func request(srv net.Listener, done chan uint) {
	for {
		conn, err := srv.Accept()
		if err != nil {
			done <- 1
			return
		}
		conn.Write([]byte("Hello"))
		go func() {
			buf := make([]byte, 16)
			_, err := conn.Read(buf)
			if err != nil {
				done <- 1
			}
			fmt.Printf("RECV: %s\n", buf)
			conn.Write([]byte("BYE"))
			conn.Close()
			done <- 1
		}()
	}
}
