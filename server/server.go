// ==============================================================================
// IMPORTS AND STRUCTURE DEFINTIONS
// ==============================================================================
package main

import (
	"fmt"
	"log"
	"net"
	"strconv"
	"strings"
	"bufio"
	"crypto/sha1"
	"encoding/base64"
)

// Defining the states of the FSM parser
// It's a Go enum, giving meaningful names to states instead of numbers.
type FSMState int
const (
    READ_REQUEST_LINE FSMState = iota
    READ_HEADERS
    READ_BODY
    DONE
)

// Defining the structure of the HTTP request
type HTTPRequest struct {
    RequestLine string
    Headers     map[string]string
    Body        string
}

//==============================================================================
// MAIN FUNCTION
//==============================================================================
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
		fmt.Println("Connection established successfully.")
		go handleConnection(connection)
	}
}

//==============================================================================
// Server - Handle Connection and Upgrade
//==============================================================================
func handleConnection(conn net.Conn) {
    reader := bufio.NewReader(conn)

    // Read request line
    requestLine, _ := reader.ReadString('\n')
    if !strings.HasPrefix(requestLine, "GET ") {
        log.Println("Not a GET request")
        return
    }

    // Read headers
    headers := make(map[string]string)
    for {
        line, err := reader.ReadString('\n')
        if err != nil || line == "\r\n" || line == "\n" {
            break // end of headers
        }

        line = strings.TrimSpace(line)
        if idx := strings.Index(line, ":"); idx != -1 {
            key := strings.TrimSpace(line[:idx])
            value := strings.TrimSpace(line[idx+1:])
            headers[strings.ToLower(key)] = value
        }
    }

    // Check for WebSocket upgrade
    if headers["upgrade"] == "websocket" && strings.Contains(headers["connection"], "Upgrade") {
        secWebSocketKey := headers["sec-websocket-key"]
        if secWebSocketKey == "" {
            log.Println("Missing Sec-WebSocket-Key")
            return
        }

        acceptKey := computeAcceptKey(secWebSocketKey)

        response := fmt.Sprintf("HTTP/1.1 101 Switching Protocols\r\n"+
            "Upgrade: websocket\r\n"+
            "Connection: Upgrade\r\n"+
            "Sec-WebSocket-Accept: %s\r\n"+
            "\r\n", acceptKey)

        conn.Write([]byte(response))

        _, err := conn.Write([]byte(response))
        if err != nil {
            log.Println("Write error:", err)
            return
        }
        log.Println("WebSocket upgrade successful on server")

    }
}

//==============================================================================
// Accept Key Computation -- Servers's
//==============================================================================
func computeAcceptKey(secWebSocketKey string) string {
    const magicGUID = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"
    h := sha1.New()
    h.Write([]byte(secWebSocketKey + magicGUID))
    return base64.StdEncoding.EncodeToString(h.Sum(nil))
}