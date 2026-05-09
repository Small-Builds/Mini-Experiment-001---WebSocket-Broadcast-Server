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

	// Building the raw HTTP request.
	request := "GET/HTTP/1.1\r\n" + 
	"Host: localhost\r\n" +
	"Connection: close\r\n" + "\r\n"

	// Converting it explicitly into byte stream so it stays ordered.
	requestBytes := []byte(request)
	
	// Writing the bytes into TCP stream
	_, err = client.Write(requestBytes)

	if err != nil {
		log.Fatalf("Write error: %v", err)
		return
	}
	fmt.Println("Request sent successfully.")

}
