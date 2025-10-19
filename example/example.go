package main

import (
	"fmt"
	"time"

	"github.com/coral/ddp"
)

func main() {

	client := ddp.NewDDPController()
	client.ConnectUDP("10.0.1.9:4048")

	// Example 1: Basic write (backward compatible)
	written, err := client.Write([]byte{128, 36, 12})
	fmt.Println("Basic write:", written, err)

	// Example 2: Write with timecode for synchronized display
	// Set display to show data 100ms in the future
	client.SetTimecode(ddp.NTPTimecodeFromDuration(100 * time.Millisecond))
	written, err = client.Write([]byte{255, 0, 0}) // Red pixel
	fmt.Println("Timecoded write:", written, err)

	// Example 3: Disable timecode and continue with normal writes
	client.DisableTimecode()
	written, err = client.Write([]byte{0, 255, 0}) // Green pixel
	fmt.Println("Normal write:", written, err)

	// Keep program running
	channel := make(chan int)
	<-channel

}
