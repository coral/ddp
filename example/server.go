package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/coral/ddp"
)

func main() {
	// Create a new DDP server
	server := ddp.NewDDPServer()

	// Register handler for default display ID (1)
	server.RegisterHandler(1, func(packet *ddp.DDPPacket, addr *net.UDPAddr) error {
		fmt.Printf("Received %d bytes from %s\n", len(packet.Data), addr)
		fmt.Printf("  ID: %d, Offset: %d, Sequence: %d\n",
			packet.Header.ID,
			packet.Header.Offset,
			packet.Header.SequenceNumber)

		if packet.Header.F1.Timecode {
			fmt.Printf("  Timecode: 0x%08X\n", packet.Header.Timecode)
		}

		// Display first few bytes as RGB values
		if len(packet.Data) >= 3 {
			fmt.Printf("  First pixel: R=%d G=%d B=%d\n",
				packet.Data[0], packet.Data[1], packet.Data[2])
		}

		return nil
	})

	// Register default handler for any other IDs
	server.RegisterDefaultHandler(func(packet *ddp.DDPPacket, addr *net.UDPAddr) error {
		fmt.Printf("Received packet for unhandled ID %d from %s (%d bytes)\n",
			packet.Header.ID, addr, len(packet.Data))
		return nil
	})

	// Start server
	fmt.Println("Starting DDP server on port 4048...")
	go func() {
		if err := server.Listen(""); err != nil {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	fmt.Println("\nShutting down server...")
	server.Close()
}
