#!/bin/bash

# Test script to demonstrate persistent connections
echo "Testing HTTP persistent connections..."

# Start the server in the background
./server &
SERVER_PID=$!

# Wait for server to start
sleep 2

echo ""
echo "=== Testing with multiple requests on same connection ==="
echo ""

# Use netcat to send multiple requests on the same connection
{
    echo -e "GET / HTTP/1.1\r\nHost: localhost:4221\r\nConnection: keep-alive\r\n\r\n"
    sleep 1
    echo -e "GET /echo/hello HTTP/1.1\r\nHost: localhost:4221\r\nConnection: keep-alive\r\n\r\n"
    sleep 1
    echo -e "GET /echo/world HTTP/1.1\r\nHost: localhost:4221\r\nConnection: close\r\n\r\n"
} | nc localhost 4221

echo ""
echo "=== Server logs should show 'Keeping connection alive' messages ==="
echo ""

# Stop the server
kill $SERVER_PID
wait $SERVER_PID 2>/dev/null

echo "Test completed!"
