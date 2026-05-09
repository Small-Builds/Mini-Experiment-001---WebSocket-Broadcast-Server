package main

import(
	"fmt"
	"log"
	"net"
)

func main(){

	// Connect to TCP server
	client, error := net.Dial("tcp", "localhost:8000")
	
	if error != nil{
		log.Fatalf("Connection failed: %v",error)
	}
	
	defer client.Close()
	
	// Print successful connection.
	fmt.Println("Successfully connected to the server !")

	// Defining the message to be sent.
	message := "Experimenting with sending raw bytes.\n"

	// Converting and writing message bytes into TCP stream
	_ , error = client.Write([]byte(message))

	if error != nil{
		log.Fatalf("Failed to send message: %v",error)
	}

	fmt.Println("Message sent to server")

}