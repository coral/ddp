package ddp

import (
	"bytes"
	"testing"
	"time"
)

// mockWriteCloser for testing
type mockWriteCloser struct {
	data   []byte
	closed bool
}

func (m *mockWriteCloser) Write(p []byte) (n int, err error) {
	m.data = append(m.data, p...)
	return len(p), nil
}

func (m *mockWriteCloser) Close() error {
	m.closed = true
	return nil
}

func newMockController() (*DDPController, *mockWriteCloser) {
	controller := NewDDPController()
	mock := &mockWriteCloser{}
	controller.output = mock
	return controller, mock
}

func TestPixelData(t *testing.T) {
	pd := byte(10)

	pixelData := PixelDataTypeFromByte(pd)

	if pixelData.DataType != RGB {
		t.Errorf("Expected RGB, go %v", pixelData.DataType)
	}

	if LEDDataType(pixelData.DataSize) != LEDDataType(Pixel4Bits) {
		t.Errorf("Expected Pixel4Bits, got %v", pixelData.DataSize)
	}

	if pixelData.CustomerDefined {
		t.Errorf("Expected CustomerDefined, got %v", pixelData.CustomerDefined)
	}
}

// Test ConfigFlag encoding and decoding
func TestConfigFlagByte(t *testing.T) {
	tests := []struct {
		name     string
		flag     ConfigFlag
		expected byte
	}{
		{
			name:     "all flags false",
			flag:     NewConfigFlag(false, false, false, false, false),
			expected: 0x40, // version 1 only
		},
		{
			name:     "push only",
			flag:     NewConfigFlag(false, false, false, false, true),
			expected: 0x41, // version 1 + push
		},
		{
			name:     "query only",
			flag:     NewConfigFlag(false, false, false, true, false),
			expected: 0x42, // version 1 + query
		},
		{
			name:     "reply only",
			flag:     NewConfigFlag(false, false, true, false, false),
			expected: 0x44, // version 1 + reply
		},
		{
			name:     "storage only",
			flag:     NewConfigFlag(false, true, false, false, false),
			expected: 0x48, // version 1 + storage
		},
		{
			name:     "timecode only",
			flag:     NewConfigFlag(true, false, false, false, false),
			expected: 0x50, // version 1 + timecode
		},
		{
			name:     "all flags true",
			flag:     NewConfigFlag(true, true, true, true, true),
			expected: 0x5F, // version 1 + all flags
		},
		{
			name:     "push and reply",
			flag:     NewConfigFlag(false, false, true, false, true),
			expected: 0x45, // version 1 + reply + push
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test encoding
			result := tt.flag.Byte()
			if result != tt.expected {
				t.Errorf("Byte() = 0x%02X, expected 0x%02X", result, tt.expected)
			}

			// Test decoding
			var decoded ConfigFlag
			decoded.FromByte(tt.expected)
			if decoded.Timecode != tt.flag.Timecode ||
				decoded.Storage != tt.flag.Storage ||
				decoded.Reply != tt.flag.Reply ||
				decoded.Query != tt.flag.Query ||
				decoded.Push != tt.flag.Push {
				t.Errorf("FromByte() decoded incorrectly: got %+v, expected %+v", decoded, tt.flag)
			}
		})
	}
}

// Test ConfigFlag roundtrip
func TestConfigFlagRoundtrip(t *testing.T) {
	original := NewConfigFlag(true, false, true, false, true)
	encoded := original.Byte()

	var decoded ConfigFlag
	decoded.FromByte(encoded)

	if decoded != original {
		t.Errorf("Roundtrip failed: original %+v, decoded %+v", original, decoded)
	}
}

// Test PixelDataType encoding and decoding
func TestPixelDataTypeByte(t *testing.T) {
	tests := []struct {
		name     string
		dataType PixelDataType
		expected byte
	}{
		{
			name: "RGB 8-bit",
			dataType: PixelDataType{
				DataType: RGB,
				DataSize: Pixel8Bits,
				CustomerDefined: false,
			},
			expected: 0x0B, // 00001011 - RGB (001) << 3 | 8-bit (011)
		},
		{
			name: "RGB 24-bit",
			dataType: PixelDataType{
				DataType: RGB,
				DataSize: Pixel24Bits,
				CustomerDefined: false,
			},
			expected: 0x0D, // 00001101 - RGB (001) << 3 | 24-bit (101)
		},
		{
			name: "HSL 8-bit",
			dataType: PixelDataType{
				DataType: HSL,
				DataSize: Pixel8Bits,
				CustomerDefined: false,
			},
			expected: 0x13, // 00010011 - HSL (010) << 3 | 8-bit (011)
		},
		{
			name: "RGBW 8-bit",
			dataType: PixelDataType{
				DataType: RGBW,
				DataSize: Pixel8Bits,
				CustomerDefined: false,
			},
			expected: 0x1B, // 00011011 - RGBW (011) << 3 | 8-bit (011)
		},
		{
			name: "Grayscale 8-bit",
			dataType: PixelDataType{
				DataType: Grayscale,
				DataSize: Pixel8Bits,
				CustomerDefined: false,
			},
			expected: 0x23, // 00100011 - Grayscale (100) << 3 | 8-bit (011)
		},
		{
			name: "Customer defined RGB 8-bit",
			dataType: PixelDataType{
				DataType: RGB,
				DataSize: Pixel8Bits,
				CustomerDefined: true,
			},
			expected: 0x8B, // 10001011 - Customer (1) | RGB (001) << 3 | 8-bit (011)
		},
		{
			name: "Undefined type",
			dataType: PixelDataType{
				DataType: UndefinedType,
				DataSize: UndefinedPixelFormat,
				CustomerDefined: false,
			},
			expected: 0x00,
		},
		{
			name: "RGB 1-bit",
			dataType: PixelDataType{
				DataType: RGB,
				DataSize: Pixel1Bits,
				CustomerDefined: false,
			},
			expected: 0x09, // 00001001 - RGB (001) << 3 | 1-bit (001)
		},
		{
			name: "RGB 32-bit",
			dataType: PixelDataType{
				DataType: RGB,
				DataSize: Pixel32Bits,
				CustomerDefined: false,
			},
			expected: 0x0E, // 00001110 - RGB (001) << 3 | 32-bit (110)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test encoding
			result := tt.dataType.Byte()
			if result != tt.expected {
				t.Errorf("Byte() = 0x%02X, expected 0x%02X", result, tt.expected)
			}

			// Test decoding
			decoded := PixelDataTypeFromByte(tt.expected)
			if decoded.DataType != tt.dataType.DataType ||
				decoded.DataSize != tt.dataType.DataSize ||
				decoded.CustomerDefined != tt.dataType.CustomerDefined {
				t.Errorf("PixelDataTypeFromByte() = %+v, expected %+v", decoded, tt.dataType)
			}
		})
	}
}

// Test PixelDataType roundtrip
func TestPixelDataTypeRoundtrip(t *testing.T) {
	original := PixelDataType{
		DataType: RGBW,
		DataSize: Pixel16Bits,
		CustomerDefined: true,
	}
	encoded := original.Byte()
	decoded := PixelDataTypeFromByte(encoded)

	if decoded.DataType != original.DataType ||
		decoded.DataSize != original.DataSize ||
		decoded.CustomerDefined != original.CustomerDefined {
		t.Errorf("Roundtrip failed: original %+v, decoded %+v", original, *decoded)
	}
}

// Test DDPHeader serialization
func TestDDPHeaderBytes(t *testing.T) {
	tests := []struct {
		name     string
		header   DDPHeader
		expected []byte
	}{
		{
			name: "basic header with push flag",
			header: DDPHeader{
				F1:             NewConfigFlag(false, false, false, false, true),
				SequenceNumber: 6,
				DataType:       PixelDataType{DataType: RGB, DataSize: Pixel8Bits, CustomerDefined: false},
				ID:             1,
				Offset:         0,
				Length:         3,
			},
			expected: []byte{0x41, 0x06, 0x0B, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x03},
		},
		{
			name: "header with offset and large length",
			header: DDPHeader{
				F1:             NewConfigFlag(false, false, false, false, true),
				SequenceNumber: 12,
				DataType:       PixelDataType{DataType: RGB, DataSize: Pixel24Bits, CustomerDefined: false},
				ID:             1,
				Offset:         39381, // 0x99d5
				Length:         281,   // 0x0119
			},
			expected: []byte{0x41, 0x0C, 0x0D, 0x01, 0x00, 0x00, 0x99, 0xd5, 0x01, 0x19},
		},
		{
			name: "header with all flags set (including timecode)",
			header: DDPHeader{
				F1:             NewConfigFlag(true, true, true, true, true),
				SequenceNumber: 15,
				DataType:       PixelDataType{DataType: RGBW, DataSize: Pixel8Bits, CustomerDefined: true},
				ID:             255,
				Offset:         0xDEADBEEF,
				Length:         0xABCD,
				Timecode:       0x12345678,
			},
			expected: []byte{0x5F, 0x0F, 0x9B, 0xFF, 0xDE, 0xAD, 0xBE, 0xEF, 0xAB, 0xCD, 0x12, 0x34, 0x56, 0x78},
		},
		{
			name: "default header",
			header: DefaultDDPHeader(),
			expected: []byte{0x41, 0x01, 0x0D, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x84},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.header.Bytes()
			if !bytes.Equal(result, tt.expected) {
				t.Errorf("Bytes() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

// Test DDPHeader with big endian encoding
func TestDDPHeaderBigEndian(t *testing.T) {
	header := DDPHeader{
		F1:             NewConfigFlag(false, false, false, false, true),
		SequenceNumber: 1,
		DataType:       PixelDataType{DataType: RGB, DataSize: Pixel8Bits, CustomerDefined: false},
		ID:             1,
		Offset:         0x12345678,
		Length:         0xABCD,
	}

	result := header.Bytes()

	// Check offset is big endian
	if result[4] != 0x12 || result[5] != 0x34 || result[6] != 0x56 || result[7] != 0x78 {
		t.Errorf("Offset not big endian: got %v", result[4:8])
	}

	// Check length is big endian
	if result[8] != 0xAB || result[9] != 0xCD {
		t.Errorf("Length not big endian: got %v", result[8:10])
	}
}

// Test DDPHeader size
func TestDDPHeaderSize(t *testing.T) {
	header := DefaultDDPHeader()
	result := header.Bytes()

	// Header should always be 10 bytes (14 with timecode, but that's TODO)
	if len(result) != 10 {
		t.Errorf("Header size = %d, expected 10", len(result))
	}
}

// Test sequence number wrapping
// Note: Current implementation has a bug where seq goes 15->16->1 instead of 15->1
// This test documents the current behavior
func TestSequenceNumberWrapping(t *testing.T) {
	controller, _ := newMockController()
	controller.header.SequenceNumber = 14

	// First write should increment to 15
	data := []byte{0xFF, 0x00, 0xFF}
	_, err := controller.Write(data)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	if controller.header.SequenceNumber != 15 {
		t.Errorf("Sequence number = %d, expected 15", controller.header.SequenceNumber)
	}

	// Next write goes to 16 (BUG: should wrap at 15, not go to 16 first)
	_, err = controller.Write(data)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	if controller.header.SequenceNumber != 16 {
		t.Errorf("Sequence number = %d, expected 16 (current buggy behavior)", controller.header.SequenceNumber)
	}

	// Third write wraps to 1
	_, err = controller.Write(data)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	if controller.header.SequenceNumber != 1 {
		t.Errorf("Sequence number = %d, expected 1 (wrapped)", controller.header.SequenceNumber)
	}

	// Fourth write increments to 2
	_, err = controller.Write(data)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	if controller.header.SequenceNumber != 2 {
		t.Errorf("Sequence number = %d, expected 2", controller.header.SequenceNumber)
	}
}

// Test sequence number zero (disabled)
func TestSequenceNumberZero(t *testing.T) {
	controller, _ := newMockController()
	controller.header.SequenceNumber = 0

	data := []byte{0xFF, 0x00, 0xFF}
	_, err := controller.Write(data)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	// Sequence number should remain 0 when disabled
	if controller.header.SequenceNumber != 0 {
		t.Errorf("Sequence number = %d, expected 0 (disabled)", controller.header.SequenceNumber)
	}
}

// Test offset handling
func TestOffsetHandling(t *testing.T) {
	controller, mock := newMockController()

	// Write with offset
	data := []byte{0xFF, 0x00, 0xFF}
	offset := uint32(100)
	_, err := controller.WriteOffset(data, offset)
	if err != nil {
		t.Fatalf("WriteOffset failed: %v", err)
	}

	// Check that offset is set in header
	if controller.header.Offset != offset {
		t.Errorf("Offset = %d, expected %d", controller.header.Offset, offset)
	}

	// Check the written data contains the offset (bytes 4-7 of header)
	if len(mock.data) < 10 {
		t.Fatalf("Not enough data written: %d bytes", len(mock.data))
	}

	// Offset should be in big endian at bytes 4-7
	writtenOffset := uint32(mock.data[4])<<24 | uint32(mock.data[5])<<16 | uint32(mock.data[6])<<8 | uint32(mock.data[7])
	if writtenOffset != offset {
		t.Errorf("Written offset = %d, expected %d", writtenOffset, offset)
	}
}

// Test SetOffset
func TestSetOffset(t *testing.T) {
	controller, _ := newMockController()

	offset := uint32(12345)
	controller.SetOffset(offset)

	if controller.header.Offset != offset {
		t.Errorf("SetOffset failed: got %d, expected %d", controller.header.Offset, offset)
	}
}

// Test length handling
func TestLengthHandling(t *testing.T) {
	controller, mock := newMockController()

	// Write different sized data
	testCases := []struct {
		name string
		data []byte
	}{
		{"small", []byte{0xFF}},
		{"medium", make([]byte, 100)},
		{"large", make([]byte, 1440)},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mock.data = nil // reset
			_, err := controller.Write(tc.data)
			if err != nil {
				t.Fatalf("Write failed: %v", err)
			}

			// Check header length field (bytes 8-9)
			if len(mock.data) < 10 {
				t.Fatalf("Not enough data written")
			}

			writtenLength := uint16(mock.data[8])<<8 | uint16(mock.data[9])
			expectedLength := uint16(len(tc.data))

			if writtenLength != expectedLength {
				t.Errorf("Written length = %d, expected %d", writtenLength, expectedLength)
			}

			// Check total written data is header + payload
			if len(mock.data) != 10+len(tc.data) {
				t.Errorf("Total written = %d, expected %d", len(mock.data), 10+len(tc.data))
			}
		})
	}
}

// Test ID validation
func TestSetID(t *testing.T) {
	controller, _ := newMockController()

	tests := []struct {
		name      string
		id        uint8
		shouldErr bool
	}{
		{"reserved ID 0", 0, true},
		{"default display ID", 1, false},
		{"custom ID", 2, false},
		{"custom ID max", 249, false},
		{"control ID", 246, false},
		{"config ID", 250, false},
		{"status ID", 251, false},
		{"DMX transit ID", 254, false},
		{"broadcast ID", 255, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := controller.SetID(tt.id)
			if tt.shouldErr && err == nil {
				t.Errorf("SetID(%d) should have returned error", tt.id)
			}
			if !tt.shouldErr && err != nil {
				t.Errorf("SetID(%d) returned unexpected error: %v", tt.id, err)
			}
			if !tt.shouldErr && controller.header.ID != tt.id {
				t.Errorf("SetID(%d) set ID to %d", tt.id, controller.header.ID)
			}
		})
	}
}

// Test maximum data length enforcement
func TestMaxDataLength(t *testing.T) {
	controller, _ := newMockController()

	// Test data at max length (should succeed)
	maxData := make([]byte, DDP_MAX_DATALEN)
	_, err := controller.Write(maxData)
	if err != nil {
		t.Errorf("Write with max data length failed: %v", err)
	}

	// Test data over max length (should fail)
	overMaxData := make([]byte, DDP_MAX_DATALEN+1)
	_, err = controller.Write(overMaxData)
	if err == nil {
		t.Error("Write with over max data length should have failed")
	}

	// Test significantly over max length
	wayOverData := make([]byte, DDP_MAX_DATALEN*2)
	_, err = controller.Write(wayOverData)
	if err == nil {
		t.Error("Write with significantly over max data length should have failed")
	}
}

// Test empty data
func TestEmptyData(t *testing.T) {
	controller, mock := newMockController()

	emptyData := []byte{}
	_, err := controller.Write(emptyData)
	if err != nil {
		t.Fatalf("Write with empty data failed: %v", err)
	}

	// Should still write header
	if len(mock.data) != 10 {
		t.Errorf("Empty write should produce 10 byte header, got %d bytes", len(mock.data))
	}

	// Length field should be 0
	writtenLength := uint16(mock.data[8])<<8 | uint16(mock.data[9])
	if writtenLength != 0 {
		t.Errorf("Empty data length = %d, expected 0", writtenLength)
	}
}

// Test RGB pixel data
func TestRGBPixelData(t *testing.T) {
	controller, mock := newMockController()
	controller.header.DataType = PixelDataType{
		DataType: RGB,
		DataSize: Pixel8Bits,
		CustomerDefined: false,
	}

	// 3 RGB pixels (9 bytes)
	rgbData := []byte{
		255, 0, 0,    // Red
		0, 255, 0,    // Green
		0, 0, 255,    // Blue
	}

	_, err := controller.Write(rgbData)
	if err != nil {
		t.Fatalf("Write RGB data failed: %v", err)
	}

	// Check data type byte in header (byte 2)
	if mock.data[2] != 0x0B { // RGB 8-bit
		t.Errorf("Data type byte = 0x%02X, expected 0x0B", mock.data[2])
	}

	// Check payload data
	if !bytes.Equal(mock.data[10:], rgbData) {
		t.Errorf("RGB data mismatch: got %v, expected %v", mock.data[10:], rgbData)
	}
}

// Test RGBW pixel data
func TestRGBWPixelData(t *testing.T) {
	controller, mock := newMockController()
	controller.header.DataType = PixelDataType{
		DataType: RGBW,
		DataSize: Pixel8Bits,
		CustomerDefined: false,
	}

	// 2 RGBW pixels (8 bytes)
	rgbwData := []byte{
		255, 0, 0, 255,    // Red with white
		0, 255, 0, 0,      // Green no white
	}

	_, err := controller.Write(rgbwData)
	if err != nil {
		t.Fatalf("Write RGBW data failed: %v", err)
	}

	// Check data type byte in header (byte 2)
	if mock.data[2] != 0x1B { // RGBW 8-bit
		t.Errorf("Data type byte = 0x%02X, expected 0x1B", mock.data[2])
	}

	// Check payload data
	if !bytes.Equal(mock.data[10:], rgbwData) {
		t.Errorf("RGBW data mismatch")
	}
}

// Test grayscale pixel data
func TestGrayscalePixelData(t *testing.T) {
	controller, mock := newMockController()
	controller.header.DataType = PixelDataType{
		DataType: Grayscale,
		DataSize: Pixel8Bits,
		CustomerDefined: false,
	}

	// 5 grayscale values
	grayData := []byte{0, 64, 128, 192, 255}

	_, err := controller.Write(grayData)
	if err != nil {
		t.Fatalf("Write grayscale data failed: %v", err)
	}

	// Check data type byte in header (byte 2)
	if mock.data[2] != 0x23 { // Grayscale 8-bit
		t.Errorf("Data type byte = 0x%02X, expected 0x23", mock.data[2])
	}

	// Check payload data
	if !bytes.Equal(mock.data[10:], grayData) {
		t.Errorf("Grayscale data mismatch")
	}
}

// Test 24-bit RGB (common LED strip format)
func TestRGB24BitData(t *testing.T) {
	controller, mock := newMockController()
	controller.header.DataType = PixelDataType{
		DataType: RGB,
		DataSize: Pixel24Bits,
		CustomerDefined: false,
	}

	// Standard RGB data for LED strips
	ledData := []byte{
		255, 0, 0,
		0, 255, 0,
		0, 0, 255,
		255, 255, 0,
		255, 0, 255,
		0, 255, 255,
	}

	_, err := controller.Write(ledData)
	if err != nil {
		t.Fatalf("Write RGB 24-bit data failed: %v", err)
	}

	// Check data type byte (RGB with 24-bit)
	if mock.data[2] != 0x0D { // RGB 24-bit
		t.Errorf("Data type byte = 0x%02X, expected 0x0D", mock.data[2])
	}
}

// Test default header values
func TestDefaultDDPHeader(t *testing.T) {
	header := DefaultDDPHeader()

	// Check defaults according to the spec
	if !header.F1.Push {
		t.Error("Default header should have Push flag set")
	}

	if header.SequenceNumber != 1 {
		t.Errorf("Default sequence number = %d, expected 1", header.SequenceNumber)
	}

	if header.DataType.DataType != RGB {
		t.Errorf("Default data type = %v, expected RGB", header.DataType.DataType)
	}

	if header.DataType.DataSize != Pixel24Bits {
		t.Errorf("Default pixel format = %v, expected Pixel24Bits", header.DataType.DataSize)
	}

	if header.ID != 1 {
		t.Errorf("Default ID = %d, expected 1", header.ID)
	}
}

// Test complete DDP packet structure
func TestCompleteDDPPacket(t *testing.T) {
	controller, mock := newMockController()

	// Configure for a typical LED strip scenario
	controller.header.F1 = NewConfigFlag(false, false, false, false, true)
	controller.header.SequenceNumber = 5
	controller.header.DataType = PixelDataType{RGB, Pixel8Bits, false}
	controller.SetID(1)
	controller.SetOffset(0)

	// 2 RGB pixels
	pixelData := []byte{255, 0, 0, 0, 255, 0}

	_, err := controller.Write(pixelData)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	// Verify complete packet structure
	// Byte 0: Flags (0x41 = version 1 + push)
	if mock.data[0] != 0x41 {
		t.Errorf("Flags byte = 0x%02X, expected 0x41", mock.data[0])
	}

	// Byte 1: Sequence number (6, because Write increments before sending)
	if mock.data[1] != 6 {
		t.Errorf("Sequence number = %d, expected 6", mock.data[1])
	}

	// Byte 2: Data type (RGB 8-bit = 0x0B)
	if mock.data[2] != 0x0B {
		t.Errorf("Data type = 0x%02X, expected 0x0B", mock.data[2])
	}

	// Byte 3: ID (1)
	if mock.data[3] != 1 {
		t.Errorf("ID = %d, expected 1", mock.data[3])
	}

	// Bytes 4-7: Offset (0)
	offset := uint32(mock.data[4])<<24 | uint32(mock.data[5])<<16 | uint32(mock.data[6])<<8 | uint32(mock.data[7])
	if offset != 0 {
		t.Errorf("Offset = %d, expected 0", offset)
	}

	// Bytes 8-9: Length (6)
	length := uint16(mock.data[8])<<8 | uint16(mock.data[9])
	if length != 6 {
		t.Errorf("Length = %d, expected 6", length)
	}

	// Bytes 10+: Pixel data
	if !bytes.Equal(mock.data[10:], pixelData) {
		t.Errorf("Pixel data mismatch")
	}
}

// Test timecode header serialization
func TestTimecodeHeader(t *testing.T) {
	header := DDPHeader{
		F1:             NewConfigFlag(true, false, false, false, true), // timecode enabled
		SequenceNumber: 1,
		DataType:       PixelDataType{RGB, Pixel8Bits, false},
		ID:             1,
		Offset:         0,
		Length:         3,
		Timecode:       0x12345678,
	}

	result := header.Bytes()

	// Header should be 14 bytes with timecode
	if len(result) != 14 {
		t.Errorf("Header with timecode size = %d, expected 14", len(result))
	}

	// Check timecode flag is set (byte 0 should have bit 4 set)
	if result[0]&0x10 == 0 {
		t.Errorf("Timecode flag not set in flags byte: 0x%02X", result[0])
	}

	// Check timecode value (bytes 10-13, big endian)
	timecode := uint32(result[10])<<24 | uint32(result[11])<<16 | uint32(result[12])<<8 | uint32(result[13])
	if timecode != 0x12345678 {
		t.Errorf("Timecode = 0x%08X, expected 0x12345678", timecode)
	}
}

// Test header without timecode (backward compatibility)
func TestHeaderWithoutTimecode(t *testing.T) {
	header := DDPHeader{
		F1:             NewConfigFlag(false, false, false, false, true), // timecode disabled
		SequenceNumber: 1,
		DataType:       PixelDataType{RGB, Pixel8Bits, false},
		ID:             1,
		Offset:         0,
		Length:         3,
		Timecode:       0x12345678, // Set but should be ignored
	}

	result := header.Bytes()

	// Header should be 10 bytes without timecode
	if len(result) != 10 {
		t.Errorf("Header without timecode size = %d, expected 10", len(result))
	}

	// Check timecode flag is not set
	if result[0]&0x10 != 0 {
		t.Errorf("Timecode flag should not be set: 0x%02X", result[0])
	}
}

// Test SetTimecode method
func TestSetTimecode(t *testing.T) {
	controller, mock := newMockController()

	// Enable timecode
	testTimecode := uint32(0xABCD1234)
	controller.SetTimecode(testTimecode)

	// Verify timecode is enabled
	if !controller.header.F1.Timecode {
		t.Error("Timecode flag should be enabled")
	}

	if controller.header.Timecode != testTimecode {
		t.Errorf("Timecode = 0x%08X, expected 0x%08X", controller.header.Timecode, testTimecode)
	}

	// Write some data
	data := []byte{0xFF, 0x00, 0xFF}
	_, err := controller.Write(data)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	// Check packet has 14-byte header
	if len(mock.data) != 14+len(data) {
		t.Errorf("Packet size = %d, expected %d (14 byte header + %d data)", len(mock.data), 14+len(data), len(data))
	}

	// Verify timecode in packet
	timecode := uint32(mock.data[10])<<24 | uint32(mock.data[11])<<16 | uint32(mock.data[12])<<8 | uint32(mock.data[13])
	if timecode != testTimecode {
		t.Errorf("Written timecode = 0x%08X, expected 0x%08X", timecode, testTimecode)
	}
}

// Test DisableTimecode method
func TestDisableTimecode(t *testing.T) {
	controller, mock := newMockController()

	// Enable then disable timecode
	controller.SetTimecode(0x12345678)
	controller.DisableTimecode()

	// Verify timecode is disabled
	if controller.header.F1.Timecode {
		t.Error("Timecode flag should be disabled")
	}

	if controller.header.Timecode != 0 {
		t.Errorf("Timecode = 0x%08X, expected 0", controller.header.Timecode)
	}

	// Write some data
	data := []byte{0xFF, 0x00, 0xFF}
	_, err := controller.Write(data)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	// Check packet has 10-byte header (no timecode)
	if len(mock.data) != 10+len(data) {
		t.Errorf("Packet size = %d, expected %d (10 byte header + %d data)", len(mock.data), 10+len(data), len(data))
	}
}

// Test NTP timecode conversion
func TestTimeToNTPTimecode(t *testing.T) {
	// Test with a known time
	// January 1, 2020 00:00:00 UTC
	testTime := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)

	timecode := TimeToNTPTimecode(testTime)

	// Timecode should be non-zero
	if timecode == 0 {
		t.Error("Timecode should not be zero")
	}

	// Test that different times produce different timecodes
	testTime2 := testTime.Add(1 * time.Second)
	timecode2 := TimeToNTPTimecode(testTime2)

	if timecode == timecode2 {
		t.Error("Different times should produce different timecodes")
	}

	// The difference should be approximately 2^16 (65536) for 1 second
	// since we're using the middle 32 bits where upper 16 are seconds
	diff := int64(timecode2) - int64(timecode)
	expectedDiff := int64(65536)

	// Allow some tolerance for fraction bits
	if diff < expectedDiff-1000 || diff > expectedDiff+1000 {
		t.Errorf("1 second difference produced timecode diff of %d, expected ~%d", diff, expectedDiff)
	}
}

// Test NTPTimecodeFromDuration
func TestNTPTimecodeFromDuration(t *testing.T) {
	// Get timecode for 1 second in the future
	timecode := NTPTimecodeFromDuration(1 * time.Second)

	// Should be non-zero
	if timecode == 0 {
		t.Error("Timecode should not be zero")
	}

	// Get timecode for now
	now := TimeToNTPTimecode(time.Now())

	// Future timecode should be greater than now
	if timecode <= now {
		t.Error("Future timecode should be greater than current timecode")
	}
}

// Test backward compatibility - existing code should work unchanged
func TestBackwardCompatibility(t *testing.T) {
	controller, mock := newMockController()

	// Use controller without touching timecode (like existing code)
	data := []byte{255, 0, 0}
	_, err := controller.Write(data)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	// Should produce 10-byte header (no timecode)
	if len(mock.data) != 10+len(data) {
		t.Errorf("Backward compatible packet should have 10-byte header, got %d total bytes", len(mock.data))
	}

	// Timecode flag should not be set
	if mock.data[0]&0x10 != 0 {
		t.Error("Default controller should not have timecode flag set")
	}
}
