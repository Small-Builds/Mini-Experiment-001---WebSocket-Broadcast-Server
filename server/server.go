package main

import (
	"fmt"
	"net"
	"log"
)

func main(){

	listener, error := net.Listen("tcp",":8000")
	if error != nil {
		log.Fatalf("Error starting server: %v",error)
	}
	defer listener.Close()

	fmt.Println("Server running on port 8080")

	conn, error := listener.Accept()
	if error != nil{
		log.Fatalf("Error accepting connection: %v", error)
	} 
	defer conn.Close()

	fmt.Println("Connection established from", conn.RemoteAddr())
}