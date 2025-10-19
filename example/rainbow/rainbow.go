package main

import (
	"fmt"
	"log"
	"math"
	"time"

	"github.com/coral/ddp"
)

const (
	numPixels  = 128
	frameRate  = 60 // frames per second
	targetAddr = "127.0.0.1:4048"
)

// hsvToRGB converts HSV color space to RGB
// h: 0-360, s: 0-1, v: 0-1
// returns r, g, b: 0-255
func hsvToRGB(h, s, v float64) (uint8, uint8, uint8) {
	c := v * s
	x := c * (1 - math.Abs(math.Mod(h/60.0, 2)-1))
	m := v - c

	var r, g, b float64
	switch {
	case h < 60:
		r, g, b = c, x, 0
	case h < 120:
		r, g, b = x, c, 0
	case h < 180:
		r, g, b = 0, c, x
	case h < 240:
		r, g, b = 0, x, c
	case h < 300:
		r, g, b = x, 0, c
	default:
		r, g, b = c, 0, x
	}

	return uint8((r + m) * 255), uint8((g + m) * 255), uint8((b + m) * 255)
}

// generateRainbowFrame creates a rainbow pattern with animation offset
// offset shifts the rainbow pattern along the strip (0-360)
func generateRainbowFrame(offset float64) []byte {
	data := make([]byte, numPixels*3) // RGB = 3 bytes per pixel

	for i := 0; i < numPixels; i++ {
		// Calculate hue for this pixel (spread across full spectrum)
		hue := math.Mod(offset+(float64(i)/float64(numPixels)*360), 360)

		// Full saturation and value for vibrant colors
		r, g, b := hsvToRGB(hue, 1.0, 1.0)

		// DDP expects RGB format
		data[i*3] = r
		data[i*3+1] = g
		data[i*3+2] = b
	}

	return data
}

func main() {
	// Create DDP controller
	controller := ddp.NewDDPController()

	// Connect to DDP server
	err := controller.ConnectUDP(targetAddr)
	if err != nil {
		log.Fatalf("Failed to connect to %s: %v", targetAddr, err)
	}
	defer controller.Close()

	fmt.Printf("Connected to DDP server at %s\n", targetAddr)
	fmt.Printf("Sending rainbow animation to %d pixels at %d FPS\n", numPixels, frameRate)
	fmt.Println("Press Ctrl+C to stop")

	// Animation parameters
	frameDuration := time.Second / frameRate
	ticker := time.NewTicker(frameDuration)
	defer ticker.Stop()

	offset := 0.0
	cycleSpeed := 2.0 // degrees per frame

	for range ticker.C {
		// Generate rainbow frame with current offset
		pixelData := generateRainbowFrame(offset)

		// Send to DDP server
		_, err := controller.Write(pixelData)
		if err != nil {
			log.Printf("Error sending data: %v", err)
			continue
		}

		// Update offset for next frame (creates cycling animation)
		offset = math.Mod(offset+cycleSpeed, 360)
	}
}
