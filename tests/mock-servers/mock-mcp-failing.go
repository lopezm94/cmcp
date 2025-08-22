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
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Error: No failure mode specified\n")
		os.Exit(1)
	}

	failureMode := os.Args[1]

	switch failureMode {
	case "exit-1":
		// Immediate exit with error code
		fmt.Fprintf(os.Stderr, "Fatal error: Server failed to start\n")
		os.Exit(1)

	case "timeout":
		// Hang forever (simulate timeout)
		select {}

	case "invalid-json":
		// Send invalid JSON response
		fmt.Println("{invalid json: true}")
		time.Sleep(50 * time.Millisecond)
		os.Exit(1)

	case "crash-after-init":
		// Start normally then crash
		scanner := bufio.NewScanner(os.Stdin)
		writer := bufio.NewWriter(os.Stdout)

		if scanner.Scan() {
			var req JSONRPCRequest
			if err := json.Unmarshal(scanner.Bytes(), &req); err == nil {
				if req.Method == "initialize" {
					response := JSONRPCResponse{
						JSONRPC: "2.0",
						ID:      req.ID,
						Result: map[string]interface{}{
							"protocolVersion": "0.1.0",
							"capabilities":    map[string]interface{}{},
							"serverInfo": map[string]interface{}{
								"name":    "mock-mcp-failing",
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
		// Crash after initialization
		time.Sleep(100 * time.Millisecond)
		fmt.Fprintf(os.Stderr, "Server crashed unexpectedly\n")
		os.Exit(1)

	case "permission-denied":
		// Simulate permission denied error
		fmt.Fprintf(os.Stderr, "Error: Permission denied\n")
		os.Exit(126)

	case "not-found":
		// Simulate command not found
		fmt.Fprintf(os.Stderr, "Error: Command not found\n")
		os.Exit(127)

	case "slow-start":
		// Take a long time to start
		time.Sleep(2 * time.Second)
		fmt.Fprintf(os.Stderr, "Server startup timeout\n")
		os.Exit(1)

	default:
		fmt.Fprintf(os.Stderr, "Unknown failure mode: %s\n", failureMode)
		os.Exit(1)
	}
}