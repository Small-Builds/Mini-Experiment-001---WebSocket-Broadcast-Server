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
// Function for parsing HTTP requests
//==============================================================================
func parseHTTP(data []byte) HTTPRequest {
	
	// Initializing the starting state
	state := READ_REQUEST_LINE
	
	// Creating the object for receving the requests
	request := HTTPRequest{Headers: make(map[string]string)}

	i := 0
	lineStart := 0
	contentLength := 0

	for i< len(data) {
		// Detect \r\n
        isCRLF := i+1 < len(data) && data[i] == '\r' && data[i+1] == '\n'
        // Detect \r\n\r\n
        isDoubleCRLF := isCRLF && i+3 < len(data) && data[i+2] == '\r' && data[i+3] == '\n'

		// Using switch case to define transitions among states
		switch state {
        case READ_REQUEST_LINE:
            if isCRLF {
                request.RequestLine = string(data[lineStart:i])
                i += 2
                lineStart = i
                state = READ_HEADERS
            } else {
                i++
            }

        case READ_HEADERS:
            if isDoubleCRLF {
                // Store last header if any
                if i > lineStart {
                    line := string(data[lineStart:i])
                    parts := strings.SplitN(line, ": ", 2)
                    if len(parts) == 2 {
                        request.Headers[parts[0]] = parts[1]
                        if parts[0] == "Content-Length" {
                            contentLength, _ = strconv.Atoi(parts[1])
                        }
                    }
                }
                i += 4 // skip \r\n\r\n
                lineStart = i
                state = READ_BODY
            } else if isCRLF {
                line := string(data[lineStart:i])
                parts := strings.SplitN(line, ": ", 2)
                if len(parts) == 2 {
                    request.Headers[parts[0]] = parts[1]
                    if parts[0] == "Content-Length" {
                        contentLength, _ = strconv.Atoi(parts[1])
                    }
                }
                i += 2
                lineStart = i
            } else {
                i++
            }

        case READ_BODY:
            end := lineStart + contentLength
            if end > len(data) {
                end = len(data) // guard against incomplete reads
            }
            request.Body = string(data[lineStart:end])
            state = DONE
            i = end

        case DONE:
            break
        }
    }

    return request
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