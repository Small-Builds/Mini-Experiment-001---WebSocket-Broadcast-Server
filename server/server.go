package main

import (
	"fmt"
	"log"
	"net"
)

func main() {

	// TCP Listener
	listener, err := net.Listen("tcp", ":8000")
	if err != nil {
		log.Fatalf("err starting server: %v", err)
	}
	defer listener.Close()

	fmt.Println("Server running on port 8080")

	// Loop for multiple connections

	for {
		// Accepting connections
		connection, err := listener.Accept()
		if err != nil {
			log.Fatalf("err accepting connection: %v", err)
		}
		go handle(connection)
	}
}

func handle(connection net.Conn) {
	defer connection.Close()

	buffer := make([]byte, 1024)
	for {
		bytesRead, err := connection.Read(buffer)
		if err != nil {
			log.Fatalf("Can't read from stream: %v", err)
		}
		fmt.Println(string(buffer[:bytesRead]))
	}
}
