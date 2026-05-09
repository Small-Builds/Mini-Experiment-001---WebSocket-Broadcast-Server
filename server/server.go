// ==============================================================================
// IMPORTS AND STRUCTURE DEFINTIONS
// ==============================================================================
package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"strconv"
	"strings"
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
		go handle(connection)
	}
}

//==============================================================================
// Function for creating separate GoRoutine for each connection.
//==============================================================================
func handle(connection net.Conn) {
	defer connection.Close()

	buffer := make([]byte, 1024)
	bytesRead, err := connection.Read(buffer)

	// Passing over to the parser
	HTTPRequest := parseHTTP(buffer[:bytesRead])

	if err != nil && err != io.EOF {
		log.Fatalf("Can't read from stream: %v", err)
	}
	fmt.Println(HTTPRequest)
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