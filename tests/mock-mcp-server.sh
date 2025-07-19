#!/bin/bash
# Mock MCP server that logs to a file and stays running

SERVER_NAME="${1:-mock-server}"
LOG_FILE="/tmp/mcp-${SERVER_NAME}.log"

echo "Mock MCP Server '${SERVER_NAME}' starting..." > "$LOG_FILE"
echo "PID: $$" >> "$LOG_FILE"
echo "Started at: $(date)" >> "$LOG_FILE"

# Trap signals to log shutdown
trap 'echo "Server ${SERVER_NAME} shutting down at $(date)" >> "$LOG_FILE"; exit 0' SIGTERM SIGKILL

# Keep the server running
while true; do
    echo "Heartbeat at $(date)" >> "$LOG_FILE"
    sleep 5
done