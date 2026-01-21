package quadtree

import (
	"fmt"
	"image"
	"image/draw"
	"path/filepath"

	"github.com/disintegration/imaging"
)

// Coordinate represents a tile position in the quadtree
type Coordinate struct {
	Z int // Zoom level (0-20, where 20 is highest detail)
	X int // X coordinate
	Y int // Y coordinate
}

// String returns a string representation of the coordinate
func (c Coordinate) String() string {
	return fmt.Sprintf("Z:%d X:%d Y:%d", c.Z, c.X, c.Y)
}

// TilePath returns the file system path for this coordinate
func (c Coordinate) TilePath(storageRoot string) string {
	return filepath.Join(storageRoot, fmt.Sprintf("%d", c.Z), fmt.Sprintf("%d", c.X), fmt.Sprintf("%d.webp", c.Y))
}

// ParentCoordinate returns the parent tile coordinate at Z-1
func (c Coordinate) ParentCoordinate() Coordinate {
	return Coordinate{
		Z: c.Z - 1,
		X: c.X / 2,
		Y: c.Y / 2,
	}
}

// Neighbors returns the 3 neighboring coordinates needed to complete a "Group of 4"
// For a tile at (X, Y), we need: (X+1, Y), (X, Y+1), (X+1, Y+1)
// But we normalize to the top-left corner of the 2x2 group
func (c Coordinate) GetGroupOf4() []Coordinate {
	// Normalize to top-left corner of 2x2 group
	baseX := (c.X / 2) * 2
	baseY := (c.Y / 2) * 2

	return []Coordinate{
		{Z: c.Z, X: baseX, Y: baseY},         // Top-left
		{Z: c.Z, X: baseX + 1, Y: baseY},     // Top-right
		{Z: c.Z, X: baseX, Y: baseY + 1},     // Bottom-left
		{Z: c.Z, X: baseX + 1, Y: baseY + 1}, // Bottom-right
	}
}

// QuadtreeManager handles tile merging and parent generation
type QuadtreeManager struct {
	storageRoot string
	tileSize    int
}

// NewQuadtreeManager creates a new quadtree manager
func NewQuadtreeManager(storageRoot string, tileSize int) *QuadtreeManager {
	return &QuadtreeManager{
		storageRoot: storageRoot,
		tileSize:    tileSize,
	}
}

// CheckAndMerge checks if a "Group of 4" is complete, and if so, merges them into the parent level
// Returns the parent coordinate if merged, nil otherwise
func (qm *QuadtreeManager) CheckAndMerge(coord Coordinate, storage Storage) (*Coordinate, error) {
	// Can't merge if we're already at zoom level 0
	if coord.Z == 0 {
		return nil, nil
	}

	// Get all 4 tiles in the group
	group := coord.GetGroupOf4()

	// Check if all 4 tiles exist
	for _, c := range group {
		exists, err := storage.TileExists(c)
		if err != nil {
			return nil, fmt.Errorf("failed to check tile existence for %s: %w", c.String(), err)
		}
		if !exists {
			// Group is not complete yet
			return nil, nil
		}
	}

	// All 4 tiles exist - merge them!
	parentCoord, err := qm.mergeGroup(group, storage)
	if err != nil {
		return nil, fmt.Errorf("failed to merge group: %w", err)
	}

	return parentCoord, nil
}

// mergeGroup merges 4 tiles into a single parent tile
// 1. Load all 4 tiles
// 2. Arrange them in a 2x2 grid on a 512x512 canvas
// 3. Downscale to 256x256
// 4. Save to parent level (Z-1, X/2, Y/2)
func (qm *QuadtreeManager) mergeGroup(group []Coordinate, storage Storage) (*Coordinate, error) {
	if len(group) != 4 {
		return nil, fmt.Errorf("invalid group size: expected 4, got %d", len(group))
	}

	// Create 512x512 canvas
	canvas := image.NewRGBA(image.Rect(0, 0, qm.tileSize*2, qm.tileSize*2))

	// Load and place each tile
	// Order: [0]=top-left, [1]=top-right, [2]=bottom-left, [3]=bottom-right
	positions := []image.Point{
		{X: 0, Y: 0},                     // Top-left
		{X: qm.tileSize, Y: 0},           // Top-right
		{X: 0, Y: qm.tileSize},           // Bottom-left
		{X: qm.tileSize, Y: qm.tileSize}, // Bottom-right
	}

	for i, coord := range group {
		tile, err := storage.LoadTile(coord)
		if err != nil {
			return nil, fmt.Errorf("failed to load tile %s: %w", coord.String(), err)
		}

		// Draw tile onto canvas at the appropriate position
		draw.Draw(canvas, image.Rect(
			positions[i].X, positions[i].Y,
			positions[i].X+qm.tileSize, positions[i].Y+qm.tileSize,
		), tile, image.Point{0, 0}, draw.Src)
	}

	// Downscale from 512x512 to 256x256
	merged := imaging.Resize(canvas, qm.tileSize, qm.tileSize, imaging.Lanczos)

	// Calculate parent coordinate
	parentCoord := group[0].ParentCoordinate()

	// Save merged tile
	err := storage.SaveTile(parentCoord, merged)
	if err != nil {
		return nil, fmt.Errorf("failed to save parent tile %s: %w", parentCoord.String(), err)
	}

	return &parentCoord, nil
}

// PropagateUpward recursively merges tiles upward until we reach Z=0 or can't merge
func (qm *QuadtreeManager) PropagateUpward(coord Coordinate, storage Storage) ([]Coordinate, error) {
	var generated []Coordinate

	currentCoord := coord
	for {
		parentCoord, err := qm.CheckAndMerge(currentCoord, storage)
		if err != nil {
			return generated, fmt.Errorf("merge failed at %s: %w", currentCoord.String(), err)
		}

		if parentCoord == nil {
			// Can't merge further (incomplete group or reached Z=0)
			break
		}

		generated = append(generated, *parentCoord)

		// Continue with the parent
		currentCoord = *parentCoord

		// Stop at zoom level 0
		if currentCoord.Z == 0 {
			break
		}
	}

	return generated, nil
}
