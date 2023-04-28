package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
)

const (
	DDP_PORT        = 4048
	DDP_MAX_DATALEN = 480 * 3
	DDP_ID_DISPLAY  = 1
	DDP_ID_CONFIG   = 250
	DDP_ID_STATUS   = 251
)

const (
	flagVersionMask byte = 0xc0
	flagVersion1    byte = 0x40
	flagPush        byte = 0x01
	flagQuery       byte = 0x02
	flagReply       byte = 0x04
	flagStorage     byte = 0x08
	flagTimecode    byte = 0x10
)

type ConfigFlag struct {
	Timecode bool
	Storage  bool
	Reply    bool
	Query    bool
	Push     bool
}

func (h *ConfigFlag) Byte() byte {
	var flags byte

	flags |= flagVersion1

	if h.Timecode {
		flags |= flagTimecode
	}
	if h.Storage {
		flags |= flagStorage
	}
	if h.Reply {
		flags |= flagReply
	}
	if h.Query {
		flags |= flagQuery
	}
	if h.Push {
		flags |= flagPush
	}
	return flags
}

func (h *ConfigFlag) FromByte(flags byte) {
	h.Timecode = flags&flagTimecode != 0
	h.Storage = flags&flagStorage != 0
	h.Reply = flags&flagReply != 0
	h.Query = flags&flagQuery != 0
	h.Push = flags&flagPush != 0
}

func NewConfigFlag(timecode bool, storage bool, reply bool, query bool, push bool) ConfigFlag {
	h := ConfigFlag{}
	h.Timecode = timecode
	h.Storage = storage
	h.Reply = reply
	h.Query = query
	h.Push = push

	return h
}

type LEDDataType uint8

const (
	Undefined LEDDataType = iota
	RGB
	HSL
	RGBW
	Grayscale
)

type PixelDataType struct {
	DataType        LEDDataType
	DataSize        int
	CustomerDefined bool
}

func PixelDataTypeFromByte(b byte) *PixelDataType {
	dataType := (b >> 3) & 0x07 // extract TTT bits
	dataSize := b & 0x07        // extract SSS bits
	customerDefined := b&0x80 != 0

	return &PixelDataType{
		DataType:        LEDDataType(dataType),
		DataSize:        int(dataSize),
		CustomerDefined: customerDefined,
	}
}

func (p *PixelDataType) Byte() byte {
	var dataTypeByte byte
	if p.CustomerDefined {
		dataTypeByte |= 0x80 // set C bit to 1
	}
	dataTypeByte |= byte(p.DataType) << 3   // set TTT bits
	dataTypeByte |= byte(p.DataSize) & 0x07 // set SSS bits
	return dataTypeByte
}

type DDPHeader struct {
	F1       ConfigFlag
	F2       byte
	DataType PixelDataType
	ID       byte
	Offset   uint32
	Length   uint16
}

func (d *DDPHeader) Bytes() []byte {
	var header []byte
	header = append(header, d.F1.Byte())
	header = append(header, d.F2)
	header = append(header, d.DataType.Byte())
	header = append(header, d.ID)

	// Write offset as big endian uint32
	offsetBuf := make([]byte, 4)
	binary.BigEndian.PutUint32(offsetBuf, d.Offset)
	header = append(header, offsetBuf...)

	// Write length as big endian uint16
	lengthBuf := make([]byte, 2)
	binary.BigEndian.PutUint16(lengthBuf, d.Length)
	header = append(header, lengthBuf...)

	// TODO: TIMECODE

	return header
}

func NewDDPHeader(f1 ConfigFlag, f2 byte, dataType PixelDataType, id byte, offset uint32, length uint16) DDPHeader {
	h := DDPHeader{}
	h.F1 = f1
	h.F2 = f2
	h.DataType = dataType
	h.ID = id
	h.Offset = offset
	h.Length = length

	return h
}

type DDPClient struct {
	header DDPHeader

	output io.Writer

	server *net.PacketConn
}

// func NewDDPClientFromWriterReader(output io.Writer, reader io.ReaderFrom) *DDPClient {
// 	return &DDPClient{header: DefaultDDPHeader(), output: output, input: reader}
// }

func (c *DDPClient) Send(data []byte) (int, error) {
	fmt.Println(append(c.header.Bytes(), data...))
	return c.output.Write(append(c.header.Bytes(), data...))
}

func DefaultDDPHeader() DDPHeader {
	return NewDDPHeader(NewConfigFlag(false, false, false, false, true), 0x00, PixelDataType{RGB, 2, false}, 0x01, 0, 3)
}

func NewDDPClient() DDPClient {
	return DDPClient{header: DefaultDDPHeader()}
}

func (d *DDPClient) ConnectUDP(addrString string) error {

	// Resolve UDP address
	addr, err := net.ResolveUDPAddr("udp", addrString)
	if err != nil {
		return nil, err
	}

	// Connect to UDP address
	conn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	d.output = conn

	// Listen for UDP packets
	udpServer, err := net.ListenPacket("udp", ":4048")
	if err != nil {
		log.Fatal(err)
	}
	defer udpServer.Close()

	return nil

}

func main() {

	// Open a UDP connection to port 4048
	addr, err := net.ResolveUDPAddr("udp", "10.0.1.9:4048")
	if err != nil {
		fmt.Printf("Error resolving UDP address: %v\n", err)
		return
	}
	conn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		fmt.Printf("Error dialing UDP connection: %v\n", err)
		return
	}
	defer conn.Close()

	conn.Write([]byte{65, 0, 10, 1, 0, 0, 0, 0, 0, 3, 0xFF, 0xFF, 0xFF})

	udpServer, err := net.ListenPacket("udp", ":4048")
	if err != nil {
		log.Fatal(err)
	}
	defer udpServer.Close()

	DDPClient := NewDDPClientFromWriter(conn)

	written, err := DDPClient.Send([]byte{0xFF, 0xFF, 0xFF})
	if err != nil {
		panic(err)
		return
	}
	fmt.Println(written)
	for {
		buf := make([]byte, 1024)
		_, addr, err := udpServer.ReadFrom(buf)
		if err != nil {
			continue
		}

		fmt.Println(buf[0:10], addr)
		cf := ConfigFlag{}
		cf.FromByte(buf[0])

		fmt.Println(cf)

		fmt.Println(PixelDataTypeFromByte(buf[2]))
		return
	}

}
