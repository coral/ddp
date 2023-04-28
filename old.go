// package main

// import (
// 	"encoding/binary"
// 	"fmt"
// 	"log"
// 	"net"
// )

// const (
// 	DDP_PORT           = 4048
// 	DDP_HEADER_LEN     = 10
// 	DDP_MAX_DATALEN    = 480 * 3
// 	DDP_FLAGS1_VER     = 0xc0
// 	DDP_FLAGS1_VER1    = 0x40
// 	DDP_FLAGS1_PUSH    = 0x01
// 	DDP_FLAGS1_QUERY   = 0x02
// 	DDP_FLAGS1_REPLY   = 0x04
// 	DDP_FLAGS1_STORAGE = 0x08
// 	DDP_FLAGS1_TIME    = 0x10
// 	DDP_ID_DISPLAY     = 1
// 	DDP_ID_CONFIG      = 250
// 	DDP_ID_STATUS      = 251
// )

// const (
// 	flagVersionMask byte = 0xc0
// 	flagVersion1    byte = 0x40
// 	flagPush        byte = 0x01
// 	flagQuery       byte = 0x02
// 	flagReply       byte = 0x04
// 	flagStorage     byte = 0x08
// 	flagTimecode    byte = 0x10
// )

// type ConfigFlag struct {
// 	Timecode bool
// 	Storage  bool
// 	Reply    bool
// 	Query    bool
// 	Push     bool
// }

// func (h *ConfigFlag) Byte() byte {
// 	var flags byte

// 	flags |= flagVersion1

// 	if h.Timecode {
// 		flags |= flagTimecode
// 	}
// 	if h.Storage {
// 		flags |= flagStorage
// 	}
// 	if h.Reply {
// 		flags |= flagReply
// 	}
// 	if h.Query {
// 		flags |= flagQuery
// 	}
// 	if h.Push {
// 		flags |= flagPush
// 	}
// 	return flags
// }

// func (h *ConfigFlag) FromByte(flags byte) {
// 	h.Timecode = flags&flagTimecode != 0
// 	h.Storage = flags&flagStorage != 0
// 	h.Reply = flags&flagReply != 0
// 	h.Query = flags&flagQuery != 0
// 	h.Push = flags&flagPush != 0
// }

// func NewConfigFlag(timecode bool, storage bool, reply bool, query bool, push bool) ConfigFlag {
// 	h := ConfigFlag{}
// 	h.Timecode = timecode
// 	h.Storage = storage
// 	h.Reply = reply
// 	h.Query = query
// 	h.Push = push

// 	return h
// }

// type LEDDataType uint8

// const (
// 	Undefined LEDDataType = iota
// 	RGB
// 	HSL
// 	RGBW
// 	Grayscale
// )

// type PixelDataType struct {
// 	DataType        LEDDataType
// 	DataSize        int
// 	CustomerDefined bool
// }

// func PixelDataTypeFromByte(b byte) *PixelDataType {
// 	dataType := (b >> 3) & 0x07 // extract TTT bits
// 	dataSize := b & 0x07        // extract SSS bits
// 	customerDefined := b&0x80 != 0

// 	return &PixelDataType{
// 		DataType:        LEDDataType(dataType),
// 		DataSize:        int(dataSize),
// 		CustomerDefined: customerDefined,
// 	}
// }

// func (p *PixelDataType) Byte() byte {
// 	var dataTypeByte byte
// 	if p.CustomerDefined {
// 		dataTypeByte |= 0x80 // set C bit to 1
// 	}
// 	dataTypeByte |= byte(p.DataType) << 3   // set TTT bits
// 	dataTypeByte |= byte(p.DataSize) & 0x07 // set SSS bits
// 	return dataTypeByte
// }

// type DDPHeader struct {
// 	F1       ConfigFlag
// 	F2       byte
// 	DataType PixelDataType
// 	ID       byte
// 	Offset   uint32
// 	Length   uint16
// }

// func (d *DDPHeader) Bytes() []byte {
// 	var header []byte
// 	header = append(header, d.F1.Byte())
// 	header = append(header, d.F2)
// 	header = append(header, d.DataType.Byte())
// 	header = append(header, d.ID)

// 	// Write offset as big endian uint32
// 	offsetBuf := make([]byte, 4)
// 	binary.BigEndian.PutUint32(offsetBuf, d.Offset)
// 	header = append(header, offsetBuf...)

// 	// Write length as big endian uint16
// 	lengthBuf := make([]byte, 2)
// 	binary.BigEndian.PutUint16(lengthBuf, d.Length)
// 	header = append(header, lengthBuf...)

// 	// TODO: TIMECODE

// 	return header
// }

// func NewDDPHeader(f1 ConfigFlag, f2 byte, dataType PixelDataType, id byte, offset uint32, length uint16) DDPHeader {
// 	h := DDPHeader{}
// 	h.F1 = f1
// 	h.F2 = f2
// 	h.DataType = dataType
// 	h.ID = id
// 	h.Offset = offset
// 	h.Length = length

// 	return h
// }

// func main() {

// 	f1 := NewConfigFlag(false, false, false, false, true)

// 	header := NewDDPHeader(f1, 0x00, PixelDataType{RGB, 2, false}, 0x01, 0, 3)

// 	buf := append(header.Bytes(), 0xFF, 0xFF, 128)

// 	fmt.Println(buf)

// 	// Open a UDP connection to port 4048
// 	addr, err := net.ResolveUDPAddr("udp", "10.0.1.9:4048")
// 	if err != nil {
// 		fmt.Printf("Error resolving UDP address: %v\n", err)
// 		return
// 	}
// 	conn, err := net.DialUDP("udp", nil, addr)
// 	if err != nil {
// 		fmt.Printf("Error dialing UDP connection: %v\n", err)
// 		return
// 	}
// 	defer conn.Close()

// 	// Send the DDP header over UDP
// 	_, err = conn.Write(buf)
// 	if err != nil {
// 		fmt.Printf("Error sending UDP packet: %v\n", err)
// 		return
// 	}

// 	fmt.Println("DDP header sent successfully!")

// 	udpServer, err := net.ListenPacket("udp", ":4048")
// 	if err != nil {
// 		log.Fatal(err)
// 	}
// 	defer udpServer.Close()

// 	for {
// 		buf := make([]byte, 1024)
// 		_, addr, err := udpServer.ReadFrom(buf)
// 		if err != nil {
// 			continue
// 		}

// 		fmt.Println(buf[0:10], addr)
// 		cf := ConfigFlag{}
// 		cf.FromByte(buf[0])

// 		fmt.Println(cf)

// 		fmt.Println(PixelDataTypeFromByte(buf[2]))
// 		return
// 	}

// }
