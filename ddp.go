// ddp implements parts of the Distributed Display Protocol (DDP) for sending pixel data to LED strips.
// based on the http://www.3waylabs.com/ddp/ specification.
// This is a work in progress, and is not yet fully implemented.
package ddp

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
)

const (
	DDP_PORT        = 4048
	DDP_MAX_DATALEN = 480 * 3
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
	UndefinedType LEDDataType = iota
	RGB
	HSL
	RGBW
	Grayscale
)

type LEDPixelFormat uint8

const (
	UndefinedPixelFormat LEDPixelFormat = iota
	Pixel1Bits
	Pixel4Bits
	Pixel8Bits
	Pixel16Bits
	Pixel24Bits
	Pixel32Bits
)

type PixelDataType struct {
	DataType        LEDDataType
	DataSize        LEDPixelFormat
	CustomerDefined bool
}

func PixelDataTypeFromByte(b byte) *PixelDataType {
	dataType := (b >> 3) & 0x07 // extract TTT bits
	dataSize := b & 0x07        // extract SSS bits
	customerDefined := b&0x80 != 0

	return &PixelDataType{
		DataType:        LEDDataType(dataType),
		DataSize:        LEDPixelFormat(dataSize),
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
	F1             ConfigFlag
	SequenceNumber byte
	DataType       PixelDataType
	ID             byte
	Offset         uint32
	Length         uint16
}

func (d *DDPHeader) Bytes() []byte {
	var header []byte
	header = append(header, d.F1.Byte())
	header = append(header, d.SequenceNumber)
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
	h.SequenceNumber = f2
	h.DataType = dataType
	h.ID = id
	h.Offset = offset
	h.Length = length

	return h
}

// DDPController connects to a pixel server and sends pixel data
type DDPController struct {
	header DDPHeader

	output io.WriteCloser
	server *net.PacketConn
}

func (c *DDPController) WriteOffset(data []byte, offset uint32) (int, error) {
	c.header.Offset = offset
	return c.Write(append(c.header.Bytes(), data...))
}

// Writes pixel data to the DDP server, without offset
func (c *DDPController) Write(data []byte) (int, error) {

	if len(data) > DDP_MAX_DATALEN {
		return 0, fmt.Errorf("data length %d exceeds maximum of %d", len(data), DDP_MAX_DATALEN)
	}

	// Iterate on sequence number
	if c.header.SequenceNumber != 0x00 {
		if c.header.SequenceNumber > 15 {
			c.header.SequenceNumber = 1
		} else {
			c.header.SequenceNumber++
		}
	}

	c.header.Length = uint16(len(data))
	return c.output.Write(append(c.header.Bytes(), data...))
}

func (c *DDPController) SetDefaultHeader(h DDPHeader) {
	c.header = h
}

func (c *DDPController) SetOffset(offset uint32) {
	c.header.Offset = offset
}

func (c *DDPController) SetID(id uint8) error {
	// 0 = reserved
	// 1 = default output device
	// 2=249 custom IDs, (possibly defined via JSON config)
	// 246 = JSON control (read/write)
	// 250 = JSON config  (read/write)
	// 251 = JSON status  (read only)
	// 254 = DMX transit
	// 255 = all devices

	if id == 0 {
		return errors.New("ID 0 is reserved")
	}

	c.header.ID = byte(id)

	return nil
}

func DefaultDDPHeader() DDPHeader {
	return NewDDPHeader(NewConfigFlag(false, false, false, false, true), 0x01, PixelDataType{RGB, Pixel24Bits, false}, 0x01, 0, 132)
}

func NewDDPController() *DDPController {
	return &DDPController{header: DefaultDDPHeader()}
}

func (d *DDPController) ConnectUDP(addrString string) error {

	// Resolve UDP address
	addr, err := net.ResolveUDPAddr("udp", addrString)
	if err != nil {
		return err
	}

	// Connect to UDP address
	conn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		return err
	}

	d.output = conn

	// Listen for UDP packets
	udpServer, err := net.ListenPacket("udp", fmt.Sprintf(":%d", addr.Port))
	if err != nil {
		log.Fatal(err)
	}

	d.server = &udpServer

	go d.handlePackets()

	return nil

}

func (d *DDPController) Close() error {
	d.output.Close()
	return (*d.server).Close()
}

func (d *DDPController) handlePackets() {
	buf := make([]byte, 65507)
	for {
		_, _, err := (*d.server).ReadFrom(buf)
		if err != nil {
			log.Print(err)
		}

		//fmt.Println("Received ", n, " bytes from ", addr)
		//fmt.Println(buf[0:n], (len(buf[0:n])-10)/3)
	}
}
