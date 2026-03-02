package main

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/gorilla/websocket"
)

func main() {
	token := "1c0a12d8f6e742778aa6b3bbb4499b3a712d4ce20e9844fa"

	dialer := websocket.Dialer{HandshakeTimeout: 5 * time.Second}
	conn, _, err := dialer.Dial("ws://127.0.0.1:18789", nil)
	if err != nil {
		fmt.Printf("dial error: %v\n", err)
		return
	}
	defer conn.Close()

	// Read connect.challenge
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	_, msg, _ := conn.ReadMessage()
	var challenge struct {
		Payload struct{ Nonce string `json:"nonce"` } `json:"payload"`
	}
	json.Unmarshal(msg, &challenge)

	// Send connect
	connectFrame := map[string]interface{}{
		"type": "req", "id": "c1", "method": "connect",
		"params": map[string]interface{}{
			"minProtocol": 3, "maxProtocol": 3,
			"client": map[string]interface{}{"id": "gateway-client", "version": "dev", "platform": "server", "mode": "backend"},
			"role": "operator", "scopes": []string{"operator.admin"}, "caps": []string{},
			"auth": map[string]interface{}{"token": token},
		},
	}
	data, _ := json.Marshal(connectFrame)
	conn.WriteMessage(websocket.TextMessage, data)

	// Wait for hello-ok
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	_, msg, _ = conn.ReadMessage()
	fmt.Printf("hello: %.80s...\n", string(msg))

	// Send agent request
	agentFrame := map[string]interface{}{
		"type": "req", "id": "a1", "method": "agent",
		"params": map[string]interface{}{
			"message":        "say just the word pong",
			"idempotencyKey": "test-" + fmt.Sprintf("%d", time.Now().UnixMilli()),
			"agentId":        "main",
			"sessionKey":     "agent:main:claw-mesh:wstest",
		},
	}
	data, _ = json.Marshal(agentFrame)
	conn.WriteMessage(websocket.TextMessage, data)

	// Read agent accepted response
	conn.SetReadDeadline(time.Now().Add(10 * time.Second))
	_, msg, err = conn.ReadMessage()
	if err != nil {
		fmt.Printf("agent read error: %v\n", err)
		return
	}
	fmt.Printf("agent response: %s\n", string(msg))

	var agentResp struct {
		Payload struct {
			RunID  string `json:"runId"`
			Status string `json:"status"`
		} `json:"payload"`
	}
	json.Unmarshal(msg, &agentResp)
	runId := agentResp.Payload.RunID
	fmt.Printf("runId: %s, status: %s\n", runId, agentResp.Payload.Status)

	// Send agent.wait
	waitFrame := map[string]interface{}{
		"type": "req", "id": "w1", "method": "agent.wait",
		"params": map[string]interface{}{
			"runId":     runId,
			"timeoutMs": 60000,
		},
	}
	data, _ = json.Marshal(waitFrame)
	conn.WriteMessage(websocket.TextMessage, data)

	// Read all messages until we get the wait response
	for i := 0; i < 20; i++ {
		conn.SetReadDeadline(time.Now().Add(120 * time.Second))
		_, msg, err = conn.ReadMessage()
		if err != nil {
			fmt.Printf("read error at msg %d: %v\n", i, err)
			return
		}
		var frame struct {
			Type  string          `json:"type"`
			ID    string          `json:"id"`
			Event string          `json:"event"`
			Ok    bool            `json:"ok"`
			Payload json.RawMessage `json:"payload"`
		}
		json.Unmarshal(msg, &frame)
		fmt.Printf("msg %d [type=%s id=%s event=%s]: %.200s\n", i, frame.Type, frame.ID, frame.Event, string(msg))

		if frame.Type == "res" && frame.ID == "w1" {
			fmt.Printf("\nagent.wait result: %s\n", string(frame.Payload))
			break
		}
	}
}
