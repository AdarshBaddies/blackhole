package quadtree

import (
	"bytes"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"

	"github.com/disintegration/imaging"
)

// Storage interface defines methods for tile persistence
type Storage interface {
	SaveTile(coord Coordinate, img image.Image) error
	LoadTile(coord Coordinate) (image.Image, error)
	TileExists(coord Coordinate) (bool, error)
	DeleteTile(coord Coordinate) error
}

// FileSystemStorage implements Storage using flat-file system
type FileSystemStorage struct {
	storageRoot string
}

// NewFileSystemStorage creates a new filesystem-based storage
func NewFileSystemStorage(storageRoot string) *FileSystemStorage {
	return &FileSystemStorage{
		storageRoot: storageRoot,
	}
}

// SaveTile saves a tile to the filesystem at /storage/tiles/{z}/{x}/{y}.webp
func (fs *FileSystemStorage) SaveTile(coord Coordinate, img image.Image) error {
	tilePath := coord.TilePath(fs.storageRoot)

	// Create parent directories if they don't exist
	dir := filepath.Dir(tilePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	// Create the file
	file, err := os.Create(tilePath)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", tilePath, err)
	}
	defer file.Close()

	// Encode and save as PNG (will switch to WebP when encoder is available)
	// Note: For now using PNG since golang.org/x/image/webp only has decoder
	err = imaging.Encode(file, img, imaging.PNG)
	if err != nil {
		return fmt.Errorf("failed to encode image to %s: %w", tilePath, err)
	}

	return nil
}

// LoadTile loads a tile from the filesystem
func (fs *FileSystemStorage) LoadTile(coord Coordinate) (image.Image, error) {
	tilePath := coord.TilePath(fs.storageRoot)

	file, err := os.Open(tilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open tile %s: %w", tilePath, err)
	}
	defer file.Close()

	img, err := imaging.Decode(file)
	if err != nil {
		return nil, fmt.Errorf("failed to decode image from %s: %w", tilePath, err)
	}

	return img, nil
}

// TileExists checks if a tile exists in the filesystem
func (fs *FileSystemStorage) TileExists(coord Coordinate) (bool, error) {
	tilePath := coord.TilePath(fs.storageRoot)

	_, err := os.Stat(tilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("failed to stat file %s: %w", tilePath, err)
	}

	return true, nil
}

// DeleteTile removes a tile from the filesystem
func (fs *FileSystemStorage) DeleteTile(coord Coordinate) error {
	tilePath := coord.TilePath(fs.storageRoot)

	err := os.Remove(tilePath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete tile %s: %w", tilePath, err)
	}

	return nil
}

// ScanAllTiles recursively scans the storage directory and returns all tile coordinates
func (fs *FileSystemStorage) ScanAllTiles() ([]Coordinate, error) {
	var tiles []Coordinate

	// Walk through the storage directory
	err := filepath.Walk(fs.storageRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Parse coordinate from path
		// Expected format: {storageRoot}/{z}/{x}/{y}.webp
		relPath, err := filepath.Rel(fs.storageRoot, path)
		if err != nil {
			return nil // Skip files we can't parse
		}

		var z, x, y int
		ext := filepath.Ext(relPath)
		// Remove extension for parsing
		relPath = relPath[:len(relPath)-len(ext)]

		_, err = fmt.Sscanf(relPath, "%d"+string(filepath.Separator)+"%d"+string(filepath.Separator)+"%d", &z, &x, &y)
		if err != nil {
			return nil // Skip files that don't match pattern
		}

		tiles = append(tiles, Coordinate{Z: z, X: x, Y: y})
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to scan tiles: %w", err)
	}

	return tiles, nil
}

// MemoryStorage implements Storage using in-memory map (for testing)
type MemoryStorage struct {
	tiles map[string]image.Image
}

// NewMemoryStorage creates a new in-memory storage
func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{
		tiles: make(map[string]image.Image),
	}
}

func (ms *MemoryStorage) key(coord Coordinate) string {
	return fmt.Sprintf("%d_%d_%d", coord.Z, coord.X, coord.Y)
}

func (ms *MemoryStorage) SaveTile(coord Coordinate, img image.Image) error {
	var buf bytes.Buffer
	png.Encode(&buf, img)
	copied, _ := png.Decode(&buf)
	ms.tiles[ms.key(coord)] = copied
	return nil
}

func (ms *MemoryStorage) LoadTile(coord Coordinate) (image.Image, error) {
	img, exists := ms.tiles[ms.key(coord)]
	if !exists {
		return nil, fmt.Errorf("tile not found: %s", coord.String())
	}
	return img, nil
}

func (ms *MemoryStorage) TileExists(coord Coordinate) (bool, error) {
	_, exists := ms.tiles[ms.key(coord)]
	return exists, nil
}

func (ms *MemoryStorage) DeleteTile(coord Coordinate) error {
	delete(ms.tiles, ms.key(coord))
	return nil
}
