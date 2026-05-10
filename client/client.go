package main

import (
	"bufio"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"log"
	"net"
	"strings"
    "sync"
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
    messages := []string{
        "Hello from client 1",
        "Hello from client 2",
        "Hello from client 3",
    }

    var wg sync.WaitGroup

    for _, msg := range messages {
        wg.Add(1)
        go func(message string) {
            defer wg.Done()
            runClient(message)
        }(msg)
    }

    wg.Wait()
}

func runClient(message string) {
    conn, err := net.Dial("tcp", "localhost:8000")
    if err != nil {
        log.Println("Connection failed:", err)
        return
    }
    defer conn.Close()

    client := &Client{Conn: conn, State: StateHTTP}

    for {
        switch client.State {
        case StateHTTP:
            secWebSocketKey, err := sendUpgradeRequest(client.Conn, "localhost:8000")
            if err != nil {
                log.Println("Failed to send upgrade request:", err)
                return
            }
            err = readUpgradeResponse(client.Conn, secWebSocketKey)
            if err != nil {
                log.Println("Upgrade failed:", err)
                return
            }
            client.State = StateWebSocket

        case StateWebSocket:
            frame := createFrame(message)
            _, err := client.Conn.Write(frame)
            if err != nil {
                log.Println("frame write error:", err)
                return
            }
            log.Println("Frame sent:", message)
            for {
                buf := make([]byte, 4096)
                n, err := client.Conn.Read(buf)
                if err != nil {
                    log.Println("connection closed:", err)
                    return
                }
                frame, err := parseFrame(buf[:n])
                if err != nil {
                    log.Println("frame parse error:", err)
                    return
                }
                log.Printf("Broadcast received: %s\n", string(frame.Payload))
            }
        }
    }
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

func sendUpgradeRequest(conn net.Conn, host string) (string, error) {
	key := make([]byte, 16)
	_, err := rand.Read(key)
	if err != nil {
		return "", err
	}
	secWebSocketKey := base64.StdEncoding.EncodeToString(key)

	req := fmt.Sprintf("GET /chat HTTP/1.1\r\n"+
		"Host: %s\r\n"+
		"Upgrade: websocket\r\n"+
		"Connection: Upgrade\r\n"+
		"Sec-WebSocket-Key: %s\r\n"+
		"Sec-WebSocket-Version: 13\r\n"+
		"\r\n",
		host, secWebSocketKey)

	_, err = conn.Write([]byte(req))
	if err != nil {
		return "", err
	}
	return secWebSocketKey, nil
}

func readUpgradeResponse(conn net.Conn, secWebSocketKey string) error {
	reader := bufio.NewReader(conn)

	statusLine, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read status line: %w", err)
	}
	if !strings.Contains(statusLine, "101") {
		return fmt.Errorf("unexpected status: %s", statusLine)
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

func computeAcceptKey(secWebSocketKey string) string {
	const magicGUID = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"
	h := sha1.New()
	h.Write([]byte(secWebSocketKey + magicGUID))
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
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