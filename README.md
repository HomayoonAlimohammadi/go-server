# An HTTP server from scratch in Go

This is a basic HTTP/1.1 server implementation from scratch in Go with support for persistent connections.

## Features

- **Persistent Connections (Keep-Alive)**: Connections are kept alive by default for HTTP/1.1 requests
- File serving with GET and POST operations
- Gzip compression support
- Echo endpoint
- User-Agent header inspection
- Graceful shutdown with signal handling

## Persistent Connections

The server now supports HTTP persistent connections (keep-alive):

- **HTTP/1.1**: Connections are persistent by default unless the client sends `Connection: close`
- **HTTP/1.0**: Connections are closed by default unless the client sends `Connection: keep-alive`
- Each connection can handle multiple sequential requests
- Connection timeout is set to 30 seconds for idle connections
- Server sends `Connection: keep-alive` header in responses to indicate support

### How it works:

1. When a client connects, the server keeps the connection open after processing a request
2. It loops to read the next request on the same connection
3. The connection is closed when:
   - Client sends `Connection: close` header
   - Connection times out (30 seconds)
   - An error occurs

### Testing Persistent Connections

Run the test script to see persistent connections in action:

```bash
./test_persistent.sh
```

This will send multiple requests on the same TCP connection and show the server logs indicating that connections are being kept alive.

## TODO:
- Write tests for the handlers
- Add connection pooling metrics
- Implement HTTP/2 support
