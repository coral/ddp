package main

import (
	"fmt"

	"github.com/coral/ddp"
)

func main() {

	client := ddp.NewDDPClient()
	client.ConnectUDP("10.0.1.9:4048")

	//Write one pixel
	written, err := client.Write([]byte{128, 36, 12})
	fmt.Println(written, err)

	// Keep program running
	channel := make(chan int)
	<-channel

}
