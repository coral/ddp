# DDP Examples

This directory contains example programs demonstrating the DDP library.

## Client Example

Demonstrates sending pixel data to a DDP display.

```bash
cd client
go build
./client
```

The client example shows:
- Basic pixel writes
- Timecoded writes for synchronized display
- Enabling/disabling timecode

## Server Example

Demonstrates receiving DDP packets.

```bash
cd server
go build
./server
```

The server example shows:
- Starting a DDP server on port 4048
- Registering handlers for specific IDs
- Using a default handler for unhandled IDs
- Inspecting received packets
- Graceful shutdown

## Testing Together

Terminal 1 (Server):
```bash
cd server && go run main.go
```

Terminal 2 (Client - modify IP to 127.0.0.1:4048):
```bash
cd client && go run main.go
```
