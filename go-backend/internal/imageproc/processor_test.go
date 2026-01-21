package imageproc

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"testing"
)

func TestProcessImage(t *testing.T) {
	config := &ProcessorConfig{
		TileSize:        256,
		WebPQuality:     60,
		PerlinFrequency: 8.0,
		PerlinOctaves:   4,
		PerlinOpacity:   0.6,
		PerlinAmplitude: 30.0,
	}

	processor := NewProcessor(config)

	// Create a test image (512x512 with red color)
	testImg := image.NewRGBA(image.Rect(0, 0, 512, 512))
	red := color.RGBA{R: 255, G: 0, B: 0, A: 255}
	for y := 0; y < 512; y++ {
		for x := 0; x < 512; x++ {
			testImg.Set(x, y, red)
		}
	}

	// Encode to PNG for testing
	var buf bytes.Buffer
	err := png.Encode(&buf, testImg)
	if err != nil {
		t.Fatalf("Failed to encode test image: %v", err)
	}

	// Process the image
	result, err := processor.ProcessImage(&buf)
	if err != nil {
		t.Fatalf("ProcessImage failed: %v", err)
	}

	// Verify we got data back
	if len(result) == 0 {
		t.Error("ProcessImage returned empty data")
	}

	t.Logf("Processed image size: %d bytes", len(result))
}

func TestApplyPerlinNoise(t *testing.T) {
	config := &ProcessorConfig{
		TileSize:        256,
		WebPQuality:     60,
		PerlinFrequency: 8.0,
		PerlinOctaves:   4,
		PerlinOpacity:   0.6,
		PerlinAmplitude: 30.0,
	}

	processor := NewProcessor(config)

	// Create a small test image (10x10 solid color)
	testImg := image.NewRGBA(image.Rect(0, 0, 10, 10))
	testColor := color.RGBA{R: 128, G: 128, B: 128, A: 255}
	for y := 0; y < 10; y++ {
		for x := 0; x < 10; x++ {
			testImg.Set(x, y, testColor)
		}
	}

	// Apply noise
	result := processor.applyPerlinNoise(testImg)

	// Verify the image was modified (at least some pixels should be different)
	modified := false
	for y := 0; y < 10; y++ {
		for x := 0; x < 10; x++ {
			originalColor := testImg.At(x, y)
			newColor := result.At(x, y)
			if originalColor != newColor {
				modified = true
				break
			}
		}
		if modified {
			break
		}
	}

	if !modified {
		t.Error("Perlin noise was not applied - no pixels were modified")
	}

	t.Log("Perlin noise successfully applied to image")
}

func TestClampUint8(t *testing.T) {
	processor := &Processor{}

	tests := []struct {
		input    float64
		expected uint8
	}{
		{-10.0, 0},
		{0.0, 0},
		{128.5, 128},
		{255.0, 255},
		{300.0, 255},
	}

	for _, tt := range tests {
		result := processor.clampUint8(tt.input)
		if result != tt.expected {
			t.Errorf("clampUint8(%f) = %d, expected %d", tt.input, result, tt.expected)
		}
	}
}
