package main

import (
	"fmt"
	"log"
	"net"
)

func main() {

	// Connect to TCP server
	client, err := net.Dial("tcp", "localhost:8000")

	if err != nil {
		log.Fatalf("Connection failed: %v", err)
	}

	defer client.Close()

	// Print successful connection.
	fmt.Println("Successfully connected to the server !")

	// Defining the message to be sent.
	message := "Muzan \n"

	// Converting and writing message bytes into TCP stream
	_, err = client.Write([]byte(message))

	if err != nil {
		log.Fatalf("Failed to send message: %v", err)
	}

	// Keeping the connection alive for testing
	fmt.Scanln()

}
