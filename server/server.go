package main

import (
	"bufio"
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"log"
	"net"
	"strings"
    "sync"
    "crypto/rand"
)
type FrameState int
const (
    READ_FIN_OPCODE      FrameState = iota
    READ_MASK_LENGTH
    READ_EXTENDED_LENGTH
    READ_MASKING_KEY
    READ_PAYLOAD
    FRAME_DONE
)

type WebSocketFrame struct {
    FIN     bool
    Opcode  byte
    Masked  bool
    Payload []byte
}

var (
    clients   = make(map[net.Conn]*Client)
    clientsMu sync.Mutex
)

type ProtocolState int
const (
	StateHTTP      ProtocolState = iota
	StateWebSocket
)

type Client struct {
	Conn  net.Conn
	State ProtocolState
}

func main() {
	listener, err := net.Listen("tcp", ":8000")
	if err != nil {
		log.Fatalf("err starting server: %v", err)
	}
	defer listener.Close()
	fmt.Println("Server running on port 8000")

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Println("accept error:", err)
			continue
		}
		client := &Client{Conn: conn, State: StateHTTP}
		fmt.Println("Connection established.")
		go handleConnection(client)
	}
}

func handleConnection(client *Client) {
    clientsMu.Lock()
    clients[client.Conn] = client
    clientsMu.Unlock()
    
    defer func() {
        clientsMu.Lock()
        delete(clients, client.Conn)
        clientsMu.Unlock()
        client.Conn.Close()
        log.Printf("Client disconnected. Total connected: %d\n", len(clients))
    }()
    defer client.Conn.Close()
	reader := bufio.NewReader(client.Conn)

	for {
		switch client.State {

		case StateHTTP:
			requestLine, _ := reader.ReadString('\n')
			if !strings.HasPrefix(requestLine, "GET ") {
				log.Println("Not a GET request")
				return
			}

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

			if headers["upgrade"] == "websocket" {
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
				_, err := client.Conn.Write([]byte(response))
				if err != nil {
					log.Println("Write error:", err)
					return
				}
				client.State = StateWebSocket
				log.Println("Server switched to WebSocket state")
			}

		case StateWebSocket:
            for{
            buf := make([]byte, 4096)
            n, err := client.Conn.Read(buf)
            if err != nil {
                log.Println("frame read error:", err)
                return
            }
            frame, err := parseFrame(buf[:n])
            if err != nil {
                log.Println("frame parse error:", err)
                return
            }
            msg := string(frame.Payload)
            log.Printf("Received: %s\n", msg)
            broadcast(client.Conn, msg)
            
            clientsMu.Lock()
            log.Printf("Total clients connected: %d\n", len(clients))
            clientsMu.Unlock()
            }
		}
	}
}

func computeAcceptKey(secWebSocketKey string) string {
	const magicGUID = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"
	h := sha1.New()
	h.Write([]byte(secWebSocketKey + magicGUID))
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

func parseFrame(data []byte) (WebSocketFrame, error) {
    frame := WebSocketFrame{}
    state := READ_FIN_OPCODE
    i := 0
    var payloadLen int
    var maskKey [4]byte

    for state != FRAME_DONE {
        if i >= len(data) {
            return frame, fmt.Errorf("incomplete frame")
        }

        switch state {
        case READ_FIN_OPCODE:
            frame.FIN = (data[i] & 0x80) != 0
            frame.Opcode = data[i] & 0x0F
            i++
            state = READ_MASK_LENGTH

        case READ_MASK_LENGTH:
            frame.Masked = (data[i] & 0x80) != 0
            payloadLen = int(data[i] & 0x7F)
            i++
            if payloadLen == 126 {
                state = READ_EXTENDED_LENGTH
            } else if payloadLen == 127 {
                state = READ_EXTENDED_LENGTH
            } else {
                if frame.Masked {
                    state = READ_MASKING_KEY
                } else {
                    state = READ_PAYLOAD
                }
            }

        case READ_EXTENDED_LENGTH:
            if payloadLen == 126 {
                if i+2 > len(data) {
                    return frame, fmt.Errorf("incomplete extended length")
                }
                payloadLen = int(data[i])<<8 | int(data[i+1])
                i += 2
            } else {
                if i+8 > len(data) {
                    return frame, fmt.Errorf("incomplete extended length")
                }
                payloadLen = 0
                for j := 0; j < 8; j++ {
                    payloadLen = (payloadLen << 8) | int(data[i+j])
                }
                i += 8
            }
            if frame.Masked {
                state = READ_MASKING_KEY
            } else {
                state = READ_PAYLOAD
            }

        case READ_MASKING_KEY:
            if i+4 > len(data) {
                return frame, fmt.Errorf("incomplete masking key")
            }
            copy(maskKey[:], data[i:i+4])
            i += 4
            state = READ_PAYLOAD

        case READ_PAYLOAD:
            if i+payloadLen > len(data) {
                return frame, fmt.Errorf("incomplete payload")
            }
            raw := data[i : i+payloadLen]
            if frame.Masked {
                unmasked := make([]byte, payloadLen)
                for j := 0; j < payloadLen; j++ {
                    unmasked[j] = raw[j] ^ maskKey[j%4]
                }
                frame.Payload = unmasked
            } else {
                frame.Payload = raw
            }
            i += payloadLen
            state = FRAME_DONE
        }
    }

    return frame, nil
}

func createFrame(payload string) []byte {
    data := []byte(payload)
    payloadLen := len(data)

    // Generate 4-byte masking key
    maskKey := make([]byte, 4)
    rand.Read(maskKey)

    var frame []byte

    // Byte 0: FIN=1, RSV1-3=0, Opcode=0x1 (text)
    frame = append(frame, 0x81)

    // Byte 1: MASK=1, Payload length
    if payloadLen < 126 {
        frame = append(frame, byte(0x80|payloadLen))
    } else if payloadLen <= 65535 {
        frame = append(frame, 0x80|126)
        frame = append(frame, byte(payloadLen>>8), byte(payloadLen&0xFF))
    } else {
        frame = append(frame, 0x80|127)
        for i := 7; i >= 0; i-- {
            frame = append(frame, byte(payloadLen>>(i*8)))
        }
    }

    // Masking key
    frame = append(frame, maskKey...)

    // Masked payload
    for i, b := range data {
        frame = append(frame, b^maskKey[i%4])
    }

    return frame
}

func broadcast(sender net.Conn, message string) {
    clientsMu.Lock()
    defer clientsMu.Unlock()

    frame := createFrame(message)
    for conn, client := range clients {
        if conn == sender {
            continue // skip sender
        }
        if client.State == StateWebSocket {
            conn.Write(frame)
        }
    }
}