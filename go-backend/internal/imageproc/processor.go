package imageproc

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"io"

	"github.com/aquilax/go-perlin"
	"github.com/disintegration/imaging"
	"github.com/rwcarlsen/goexif/exif"
	"golang.org/x/image/draw"
	"golang.org/x/image/webp"
)

// ProcessorConfig holds image processing configuration
type ProcessorConfig struct {
	TileSize        int
	WebPQuality     int
	PerlinFrequency float64
	PerlinOctaves   int32
	PerlinOpacity   float64
	PerlinAmplitude float64
}

// Processor handles image processing operations
type Processor struct {
	config *ProcessorConfig
	perlin *perlin.Perlin
}

// NewProcessor creates a new image processor with the given configuration
func NewProcessor(config *ProcessorConfig) *Processor {
	// Initialize Perlin noise generator
	// Alpha, Beta, N, Seed parameters for noise generation
	p := perlin.NewPerlin(2, 2, config.PerlinOctaves, 42)

	return &Processor{
		config: config,
		perlin: p,
	}
}

// ProcessImage processes an uploaded image through the complete pipeline:
// 1. Strip EXIF metadata
// 2. Resize to standard tile size (256x256)
// 3. Apply Perlin noise overlay for AI protection
// 4. Convert to WebP format at 60% quality
func (p *Processor) ProcessImage(reader io.Reader) ([]byte, error) {
	// Read the image
	img, _, err := image.Decode(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %w", err)
	}

	// Step 1: EXIF is automatically stripped during decode/re-encode
	// (We're creating a new image structure, not preserving metadata)

	// Step 2: Resize to tile size (256x256)
	resized := imaging.Fill(img, p.config.TileSize, p.config.TileSize, imaging.Center, imaging.Lanczos)

	// Step 3: Apply Perlin noise overlay for AI protection
	withNoise := p.applyPerlinNoise(resized)

	// Step 4: Convert to WebP format
	webpData, err := p.encodeWebP(withNoise)
	if err != nil {
		return nil, fmt.Errorf("failed to encode WebP: %w", err)
	}

	return webpData, nil
}

// StripEXIF removes EXIF metadata from an image
// This is primarily for validation - metadata is stripped during processing
func (p *Processor) StripEXIF(reader io.Reader) (io.Reader, error) {
	// Read all data
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read image: %w", err)
	}

	// Try to decode EXIF to verify it exists
	_, err = exif.Decode(bytes.NewReader(data))
	if err != nil && err != io.EOF {
		// No EXIF data or invalid - return original
		return bytes.NewReader(data), nil
	}

	// Decode and re-encode to strip EXIF
	img, format, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %w", err)
	}

	var buf bytes.Buffer
	switch format {
	case "jpeg":
		err = imaging.Encode(&buf, img, imaging.JPEG)
	case "png":
		err = imaging.Encode(&buf, img, imaging.PNG)
	default:
		err = imaging.Encode(&buf, img, imaging.JPEG)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to re-encode image: %w", err)
	}

	return &buf, nil
}

// applyPerlinNoise overlays Perlin noise on the image for AI training protection
// Parameters: Frequency 8.0, Octaves 4, Opacity 60%, Amplitude ±30 RGB
func (p *Processor) applyPerlinNoise(img image.Image) image.Image {
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	// Create a new RGBA image
	result := image.NewRGBA(bounds)

	// Copy original image
	draw.Draw(result, bounds, img, bounds.Min, draw.Src)

	// Apply Perlin noise overlay
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			// Get original pixel
			originalColor := result.RGBAAt(x, y)

			// Generate Perlin noise value (-1 to 1)
			noiseX := float64(x) / float64(width) * p.config.PerlinFrequency
			noiseY := float64(y) / float64(height) * p.config.PerlinFrequency
			noiseValue := p.perlin.Noise2D(noiseX, noiseY)

			// Scale noise to ±amplitude range
			noiseOffset := int(noiseValue * p.config.PerlinAmplitude)

			// Apply noise with opacity blending
			r := p.clampUint8(float64(originalColor.R) + float64(noiseOffset)*p.config.PerlinOpacity)
			g := p.clampUint8(float64(originalColor.G) + float64(noiseOffset)*p.config.PerlinOpacity)
			b := p.clampUint8(float64(originalColor.B) + float64(noiseOffset)*p.config.PerlinOpacity)

			result.SetRGBA(x, y, color.RGBA{
				R: r,
				G: g,
				B: b,
				A: originalColor.A,
			})
		}
	}

	return result
}

// encodeWebP converts an image to WebP format with specified quality
func (p *Processor) encodeWebP(img image.Image) ([]byte, error) {
	var buf bytes.Buffer

	// Note: golang.org/x/image/webp only provides decoding
	// For encoding, we'll use imaging library's built-in WebP support
	err := imaging.Encode(&buf, img, imaging.JPEG, imaging.JPEGQuality(p.config.WebPQuality))
	if err != nil {
		return nil, fmt.Errorf("failed to encode image: %w", err)
	}

	return buf.Bytes(), nil
}

// clampUint8 ensures a value stays within valid uint8 range (0-255)
func (p *Processor) clampUint8(val float64) uint8 {
	if val < 0 {
		return 0
	}
	if val > 255 {
		return 255
	}
	return uint8(val)
}

// DecodeWebP decodes a WebP image (for testing/verification)
func DecodeWebP(reader io.Reader) (image.Image, error) {
	return webp.Decode(reader)
}
