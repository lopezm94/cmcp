package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"time"
)

type JSONRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type JSONRPCResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id"`
	Result  interface{} `json:"result,omitempty"`
	Error   interface{} `json:"error,omitempty"`
}

func main() {
	scanner := bufio.NewScanner(os.Stdin)
	writer := bufio.NewWriter(os.Stdout)

	// Simulate server startup time
	time.Sleep(50 * time.Millisecond)

	// Read initialization request
	if scanner.Scan() {
		var req JSONRPCRequest
		if err := json.Unmarshal(scanner.Bytes(), &req); err == nil {
			// Respond to initialize request
			if req.Method == "initialize" {
				response := JSONRPCResponse{
					JSONRPC: "2.0",
					ID:      req.ID,
					Result: map[string]interface{}{
						"protocolVersion": "0.1.0",
						"capabilities": map[string]interface{}{
							"tools": map[string]interface{}{},
						},
						"serverInfo": map[string]interface{}{
							"name":    "mock-mcp-basic",
							"version": "1.0.0",
						},
					},
				}
				
				data, _ := json.Marshal(response)
				fmt.Fprintf(writer, "%s\n", data)
				writer.Flush()
			}
		}
	}

	// Keep running for a bit to simulate a working server
	time.Sleep(50 * time.Millisecond)
	
	// Exit cleanly
	os.Exit(0)
}