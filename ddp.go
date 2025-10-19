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
	"time"
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
	Timecode       uint32 // Optional: 32-bit NTP timecode (middle bits of 64-bit NTP time)
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

	// Add timecode if timecode flag is set
	if d.F1.Timecode {
		timecodeBuf := make([]byte, 4)
		binary.BigEndian.PutUint32(timecodeBuf, d.Timecode)
		header = append(header, timecodeBuf...)
	}

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

// ParseDDPHeader parses a DDP header from bytes
// Returns the header and the number of bytes consumed (10 or 14)
func ParseDDPHeader(data []byte) (*DDPHeader, int, error) {
	if len(data) < 10 {
		return nil, 0, errors.New("insufficient data for DDP header (need at least 10 bytes)")
	}

	header := &DDPHeader{}

	// Parse flags
	header.F1.FromByte(data[0])

	// Parse sequence number
	header.SequenceNumber = data[1]

	// Parse data type
	dt := PixelDataTypeFromByte(data[2])
	header.DataType = *dt

	// Parse ID
	header.ID = data[3]

	// Parse offset (big endian)
	header.Offset = binary.BigEndian.Uint32(data[4:8])

	// Parse length (big endian)
	header.Length = binary.BigEndian.Uint16(data[8:10])

	bytesConsumed := 10

	// Parse timecode if present
	if header.F1.Timecode {
		if len(data) < 14 {
			return nil, 0, errors.New("insufficient data for DDP header with timecode (need 14 bytes)")
		}
		header.Timecode = binary.BigEndian.Uint32(data[10:14])
		bytesConsumed = 14
	}

	return header, bytesConsumed, nil
}

// DDPPacket represents a complete received DDP packet
type DDPPacket struct {
	Header DDPHeader
	Data   []byte
}

// DDPController connects to a pixel server and sends pixel data
type DDPController struct {
	header DDPHeader

	output io.WriteCloser
	server *net.PacketConn
}

// DDPServer listens for DDP packets
type DDPServer struct {
	conn     *net.UDPConn
	handlers map[byte]PacketHandler
	running  bool
}

// PacketHandler is called when a packet is received for a specific ID
type PacketHandler func(packet *DDPPacket, addr *net.UDPAddr) error

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

// SetTimecode enables timecode and sets the value
// The timecode is the 32 middle bits of 64-bit NTP time
func (c *DDPController) SetTimecode(timecode uint32) {
	c.header.F1.Timecode = true
	c.header.Timecode = timecode
}

// DisableTimecode disables timecode support
func (c *DDPController) DisableTimecode() {
	c.header.F1.Timecode = false
	c.header.Timecode = 0
}

// TimeToNTPTimecode converts a time.Time to the 32-bit NTP timecode used by DDP
// Returns the middle 32 bits of 64-bit NTP time (16 bits seconds, 16 bits fraction)
func TimeToNTPTimecode(t time.Time) uint32 {
	// NTP epoch is January 1, 1900
	// Unix epoch is January 1, 1970
	// Difference is 70 years + 17 leap days = 2208988800 seconds
	const ntpEpochOffset = 2208988800

	unixSecs := t.Unix()
	ntpSecs := uint64(unixSecs + ntpEpochOffset)

	// Get nanoseconds and convert to NTP fraction (2^32 units per second)
	nanos := uint64(t.Nanosecond())
	ntpFrac := (nanos << 32) / 1000000000

	// Build 64-bit NTP timestamp
	ntpTime := (ntpSecs << 32) | ntpFrac

	// Return middle 32 bits (16 bits of seconds, 16 bits of fraction)
	return uint32((ntpTime >> 16) & 0xFFFFFFFF)
}

// NTPTimecodeFromDuration creates an NTP timecode from a duration relative to now
func NTPTimecodeFromDuration(d time.Duration) uint32 {
	return TimeToNTPTimecode(time.Now().Add(d))
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

	// Listen for UDP packets on any available port (for receiving replies)
	// Use port 0 to let the OS assign an available port
	udpServer, err := net.ListenPacket("udp", ":0")
	if err != nil {
		// If listening fails, just log and continue without reply support
		log.Printf("Warning: could not start listener for replies: %v", err)
		return nil
	}

	d.server = &udpServer

	go d.handlePackets()

	return nil

}

func (d *DDPController) Close() error {
	if d.output != nil {
		d.output.Close()
	}
	if d.server != nil {
		return (*d.server).Close()
	}
	return nil
}

func (d *DDPController) handlePackets() {
	buf := make([]byte, 65507)
	for {
		_, _, err := (*d.server).ReadFrom(buf)
		if err != nil {
			// Ignore closed network connection errors (happens during Close())
			if !isClosedError(err) {
				log.Print(err)
			}
			return
		}

		//fmt.Println("Received ", n, " bytes from ", addr)
		//fmt.Println(buf[0:n], (len(buf[0:n])-10)/3)
	}
}

// isClosedError checks if an error is due to a closed connection
func isClosedError(err error) bool {
	return err != nil && (err.Error() == "use of closed network connection" ||
		err.Error() == "EOF")
}

// NewDDPServer creates a new DDP server
func NewDDPServer() *DDPServer {
	return &DDPServer{
		handlers: make(map[byte]PacketHandler),
		running:  false,
	}
}

// RegisterHandler registers a handler for packets with a specific ID
func (s *DDPServer) RegisterHandler(id byte, handler PacketHandler) {
	s.handlers[id] = handler
}

// RegisterDefaultHandler registers a handler for all unhandled IDs
func (s *DDPServer) RegisterDefaultHandler(handler PacketHandler) {
	s.handlers[0xFF] = handler // Use 0xFF as special "default" key internally
}

// Listen starts the server on the specified address
// If addr is empty, listens on ":4048" (all interfaces, default DDP port)
func (s *DDPServer) Listen(addr string) error {
	if addr == "" {
		addr = fmt.Sprintf(":%d", DDP_PORT)
	}

	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return fmt.Errorf("failed to resolve address: %w", err)
	}

	conn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		return fmt.Errorf("failed to listen on UDP: %w", err)
	}

	s.conn = conn
	s.running = true

	log.Printf("DDP server listening on %s", addr)

	return s.serve()
}

// serve handles incoming packets
func (s *DDPServer) serve() error {
	buf := make([]byte, 65507) // Max UDP packet size

	for s.running {
		n, addr, err := s.conn.ReadFromUDP(buf)
		if err != nil {
			if !s.running {
				return nil // Server was closed
			}
			log.Printf("Error reading packet: %v", err)
			continue
		}

		// Parse packet in a goroutine to avoid blocking
		go s.handlePacket(buf[:n], addr)
	}

	return nil
}

// handlePacket processes a single DDP packet
func (s *DDPServer) handlePacket(data []byte, addr *net.UDPAddr) {
	// Parse header
	header, headerSize, err := ParseDDPHeader(data)
	if err != nil {
		log.Printf("Failed to parse header from %s: %v", addr, err)
		return
	}

	// Extract payload
	payload := []byte{}
	if len(data) > headerSize {
		expectedPayloadSize := int(header.Length)
		actualPayloadSize := len(data) - headerSize

		if actualPayloadSize < expectedPayloadSize {
			log.Printf("Warning: payload size mismatch from %s (expected %d, got %d)",
				addr, expectedPayloadSize, actualPayloadSize)
			payload = data[headerSize:]
		} else {
			payload = data[headerSize : headerSize+expectedPayloadSize]
		}
	}

	// Create packet
	packet := &DDPPacket{
		Header: *header,
		Data:   payload,
	}

	// Find and call handler
	handler, exists := s.handlers[header.ID]
	if !exists {
		// Try default handler
		handler, exists = s.handlers[0xFF]
		if !exists {
			log.Printf("No handler for ID %d from %s", header.ID, addr)
			return
		}
	}

	// Call handler
	if err := handler(packet, addr); err != nil {
		log.Printf("Handler error for ID %d from %s: %v", header.ID, addr, err)
	}
}

// Close stops the server
func (s *DDPServer) Close() error {
	s.running = false
	if s.conn != nil {
		return s.conn.Close()
	}
	return nil
}
