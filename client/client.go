package main

import(
	"fmt"
	"log"
	"net"
)

func main(){

	client, error := net.Dial("tcp", "localhost:8000")
	if error != nil{
		log.Fatalf("Connection failed: %v",error)
	}
	defer client.Close()
	fmt.Println("Successfully connected to the server !")
}