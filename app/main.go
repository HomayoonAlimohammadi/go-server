package main

import (
	"fmt"
	"net"
	"os"
	"time"
)

// Ensures gofmt doesn't remove the "net" and "os" imports above (feel free to remove this!)
var _ = net.Listen
var _ = os.Exit

func main() {
	// You can use print statements as follows for debugging, they'll be visible when running tests.
	fmt.Println("Logs from your program will appear here!")

	// Uncomment this block to pass the first stage
	//
	l, err := net.Listen("tcp", "0.0.0.0:4221")
	if err != nil {
		fmt.Println("Failed to bind to port 4221")
		os.Exit(1)
	}

	for {
		conn, err := l.Accept()
		if err != nil {
			fmt.Println("Error accepting connection: ", err.Error())
			os.Exit(1)
		}
		if err := conn.SetDeadline(time.Now().Add(30 * time.Second)); err != nil {
			fmt.Println("Error setting deadline: ", err.Error())
			os.Exit(1)
		}

		var b []byte
		if _, err = conn.Read(b); err != nil {
			fmt.Println("Error reading from connection: ", err.Error())
			os.Exit(1)
		}
		fmt.Println("Received data:", string(b))
		if _, err = conn.Write([]byte("HTTP/1.1 200 OK\r\n\r\n")); err != nil {
			fmt.Println("Error writing to connection: ", err.Error())
			os.Exit(1)
		}
		if err := conn.Close(); err != nil {
			fmt.Println("Error closing connection: ", err.Error())
			os.Exit(1)
		}
	}
}
