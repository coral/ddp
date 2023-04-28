// package main

// import "fmt"

// func main() {

// 	client := NewDDPClient()
// 	client.ConnectUDP("10.0.1.9:4048")
// 	written, err := client.Send([]byte{128, 36, 12})
// 	fmt.Println(written, err)

// 	channel := make(chan int)
// 	<-channel

// 