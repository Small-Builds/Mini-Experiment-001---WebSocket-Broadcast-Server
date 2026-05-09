package main

import (
	"fmt"
	"net"
	"log"
)

func main(){

	// TCP Listener
	listener, error := net.Listen("tcp",":8000")
	if error != nil {
		log.Fatalf("Error starting server: %v",error)
	}
	defer listener.Close()
	
	fmt.Println("Server running on port 8080")

	// Accepting connections
	connection, error := listener.Accept()
	if error != nil{
		log.Fatalf("Error accepting connection: %v", error)
	} 
	defer connection.Close()

	// Printing established connection's IP.
	fmt.Println("Connection established from", connection.RemoteAddr())

	// Creating buffer for incoming data
	buffer := make([]byte, 1024)

	// Read bytes from TCP stream
	bytesRead, error := connection.Read(buffer)
	
	if error != nil{
		log.Fatalf("Failed to read data: %v",error)
	}

	// Converting received bytes into string
	message := string(buffer[:bytesRead])

	fmt.Printf("Received message: %s\n",message)
}