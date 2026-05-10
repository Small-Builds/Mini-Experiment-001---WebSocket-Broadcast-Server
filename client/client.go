//==============================================================================
// IMPORTS
//==============================================================================
package main

import (
	"fmt"
	"log"
	"net"
	"bufio"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base64"
	"strings"
)

//==============================================================================
// MAIN FUNCTION
//==============================================================================
func main() {

	// Connect to TCP server
	client, err := net.Dial("tcp", "localhost:8000")
	if err != nil {
		log.Fatalf("Connection failed: %v", err)
	}
	defer client.Close()

	// Calling the upgradeRequest Function
	secWebSocketKey, err := sendUpgradeRequest(client, "localhost:8080")
    if err != nil {
        log.Fatal("Failed to send upgrade request:", err)
    }
	fmt.Println("Upgrade request sent...")

	// Calling the readUpgradeResponse function
	err = readUpgradeResponse(client, secWebSocketKey)
    if err != nil {
        log.Fatal("Upgrade failed:", err)
    }

    fmt.Println("WebSocket handshake completed successfully!")

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

//==============================================================================
// UPGRADE REQUEST FUNCTION
//==============================================================================
func sendUpgradeRequest(conn net.Conn, host string) (string, error) {
    // 1. Generate random 16 bytes and base64 encode
    key := make([]byte, 16)
    _, err := rand.Read(key)
    if err != nil {
        return "", err
    }
    secWebSocketKey := base64.StdEncoding.EncodeToString(key)

    // 2. Build HTTP Upgrade Request
    req := fmt.Sprintf("GET /chat HTTP/1.1\r\n"+
        "Host: %s\r\n"+
        "Upgrade: websocket\r\n"+
        "Connection: Upgrade\r\n"+
        "Sec-WebSocket-Key: %s\r\n"+
        "Sec-WebSocket-Version: 13\r\n"+
        "\r\n",
        host, secWebSocketKey)

    // 3. Send to server
    _, err = conn.Write([]byte(req))
    if err != nil {
        return "", err
    }

    // Return the key so we can compute expected accept key later
    return secWebSocketKey, nil
}

//==============================================================================
// Client - Read and Verify Response
//==============================================================================
func readUpgradeResponse(conn net.Conn, secWebSocketKey string) error {
    reader := bufio.NewReader(conn)

    // Read status line
    statusLine, err := reader.ReadString('\n')
    if err != nil {
        return fmt.Errorf("Failed to read status line: %s", statusLine)
    }

    // Read headers
    headers := make(map[string]string)
    for {
        line, err := reader.ReadString('\n')
        if err != nil || line == "\r\n" || line == "\n" {
            break
        }

        line = strings.TrimSpace(line)
        if idx := strings.Index(line, ":"); idx != -1 {
            key := strings.TrimSpace(line[:idx])
            value := strings.TrimSpace(line[idx+1:])
            headers[strings.ToLower(key)] = value
        }
    }

    serverAcceptKey := headers["sec-websocket-accept"]
    if serverAcceptKey == "" {
        return fmt.Errorf("missing Sec-WebSocket-Accept")
    }

    expected := computeAcceptKey(secWebSocketKey)

    if serverAcceptKey != expected {
        return fmt.Errorf("Sec-WebSocket-Accept mismatch")
    }

    log.Println("WebSocket upgrade successful on client")

    return nil
}

//==============================================================================
// Accept Key Computation -- Client's
//==============================================================================
func computeAcceptKey(secWebSocketKey string) string {
    const magicGUID = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"
    h := sha1.New()
    h.Write([]byte(secWebSocketKey + magicGUID))
    return base64.StdEncoding.EncodeToString(h.Sum(nil))
}