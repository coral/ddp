package ddp

import (
	"testing"
)

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
