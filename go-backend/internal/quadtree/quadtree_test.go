package quadtree

import (
	"image"
	"image/color"
	"testing"
)

func TestGetGroupOf4(t *testing.T) {
	tests := []struct {
		name     string
		coord    Coordinate
		expected []Coordinate
	}{
		{
			name:  "Top-left of group",
			coord: Coordinate{Z: 20, X: 0, Y: 0},
			expected: []Coordinate{
				{Z: 20, X: 0, Y: 0},
				{Z: 20, X: 1, Y: 0},
				{Z: 20, X: 0, Y: 1},
				{Z: 20, X: 1, Y: 1},
			},
		},
		{
			name:  "Bottom-right of group",
			coord: Coordinate{Z: 20, X: 101, Y: 101},
			expected: []Coordinate{
				{Z: 20, X: 100, Y: 100},
				{Z: 20, X: 101, Y: 100},
				{Z: 20, X: 100, Y: 101},
				{Z: 20, X: 101, Y: 101},
			},
		},
		{
			name:  "Random position",
			coord: Coordinate{Z: 20, X: 5, Y: 7},
			expected: []Coordinate{
				{Z: 20, X: 4, Y: 6},
				{Z: 20, X: 5, Y: 6},
				{Z: 20, X: 4, Y: 7},
				{Z: 20, X: 5, Y: 7},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			group := tt.coord.GetGroupOf4()

			if len(group) != 4 {
				t.Fatalf("Expected 4 coordinates, got %d", len(group))
			}

			for i, expected := range tt.expected {
				if group[i] != expected {
					t.Errorf("Position %d: expected %s, got %s", i, expected.String(), group[i].String())
				}
			}
		})
	}
}

func TestParentCoordinate(t *testing.T) {
	tests := []struct {
		coord    Coordinate
		expected Coordinate
	}{
		{
			coord:    Coordinate{Z: 20, X: 0, Y: 0},
			expected: Coordinate{Z: 19, X: 0, Y: 0},
		},
		{
			coord:    Coordinate{Z: 20, X: 100, Y: 100},
			expected: Coordinate{Z: 19, X: 50, Y: 50},
		},
		{
			coord:    Coordinate{Z: 20, X: 101, Y: 101},
			expected: Coordinate{Z: 19, X: 50, Y: 50},
		},
	}

	for _, tt := range tests {
		result := tt.coord.ParentCoordinate()
		if result != tt.expected {
			t.Errorf("ParentCoordinate(%s) = %s, expected %s",
				tt.coord.String(), result.String(), tt.expected.String())
		}
	}
}

func TestMergeGroup(t *testing.T) {
	storage := NewMemoryStorage()
	qm := NewQuadtreeManager("", 256)

	// Create 4 test tiles with different colors
	colors := []color.RGBA{
		{R: 255, G: 0, B: 0, A: 255},   // Red - top-left
		{R: 0, G: 255, B: 0, A: 255},   // Green - top-right
		{R: 0, G: 0, B: 255, A: 255},   // Blue - bottom-left
		{R: 255, G: 255, B: 0, A: 255}, // Yellow - bottom-right
	}

	group := []Coordinate{
		{Z: 20, X: 0, Y: 0},
		{Z: 20, X: 1, Y: 0},
		{Z: 20, X: 0, Y: 1},
		{Z: 20, X: 1, Y: 1},
	}

	// Create and save test images
	for i, coord := range group {
		img := image.NewRGBA(image.Rect(0, 0, 256, 256))
		for y := 0; y < 256; y++ {
			for x := 0; x < 256; x++ {
				img.Set(x, y, colors[i])
			}
		}
		storage.SaveTile(coord, img)
	}

	// Merge the group
	parentCoord, err := qm.mergeGroup(group, storage)
	if err != nil {
		t.Fatalf("mergeGroup failed: %v", err)
	}

	// Verify parent coordinate
	expected := Coordinate{Z: 19, X: 0, Y: 0}
	if *parentCoord != expected {
		t.Errorf("Expected parent %s, got %s", expected.String(), parentCoord.String())
	}

	// Verify parent tile exists
	exists, err := storage.TileExists(*parentCoord)
	if err != nil {
		t.Fatalf("Failed to check parent tile: %v", err)
	}
	if !exists {
		t.Error("Parent tile was not created")
	}

	// Load and verify parent tile dimensions
	parentImg, err := storage.LoadTile(*parentCoord)
	if err != nil {
		t.Fatalf("Failed to load parent tile: %v", err)
	}

	bounds := parentImg.Bounds()
	if bounds.Dx() != 256 || bounds.Dy() != 256 {
		t.Errorf("Parent tile has wrong dimensions: %dx%d, expected 256x256", bounds.Dx(), bounds.Dy())
	}

	t.Logf("Successfully merged 4 tiles into parent at %s", parentCoord.String())
}

func TestCheckAndMerge(t *testing.T) {
	storage := NewMemoryStorage()
	qm := NewQuadtreeManager("", 256)

	// Create only 3 tiles (incomplete group)
	for i := 0; i < 3; i++ {
		coord := Coordinate{Z: 20, X: i, Y: 0}
		img := image.NewRGBA(image.Rect(0, 0, 256, 256))
		storage.SaveTile(coord, img)
	}

	// Try to merge - should return nil (incomplete)
	coord := Coordinate{Z: 20, X: 0, Y: 0}
	parent, err := qm.CheckAndMerge(coord, storage)
	if err != nil {
		t.Fatalf("CheckAndMerge failed: %v", err)
	}
	if parent != nil {
		t.Error("Expected nil parent for incomplete group, got:", parent.String())
	}

	// Add the 4th tile
	coord4 := Coordinate{Z: 20, X: 1, Y: 1}
	img4 := image.NewRGBA(image.Rect(0, 0, 256, 256))
	storage.SaveTile(coord4, img4)

	// Now merge should succeed
	parent, err = qm.CheckAndMerge(coord, storage)
	if err != nil {
		t.Fatalf("CheckAndMerge failed on complete group: %v", err)
	}
	if parent == nil {
		t.Error("Expected parent coordinate, got nil")
	}

	expected := Coordinate{Z: 19, X: 0, Y: 0}
	if *parent != expected {
		t.Errorf("Expected parent %s, got %s", expected.String(), parent.String())
	}
}

func TestPropagateUpward(t *testing.T) {
	storage := NewMemoryStorage()
	qm := NewQuadtreeManager("", 256)

	// Create a complete pyramid: 4 tiles at Z=20, which should cascade up
	// This tests Z=20 -> Z=19 merge
	coords := []Coordinate{
		{Z: 20, X: 0, Y: 0},
		{Z: 20, X: 1, Y: 0},
		{Z: 20, X: 0, Y: 1},
		{Z: 20, X: 1, Y: 1},
	}

	for _, coord := range coords {
		img := image.NewRGBA(image.Rect(0, 0, 256, 256))
		storage.SaveTile(coord, img)
	}

	// Propagate upward from first tile
	generated, err := qm.PropagateUpward(coords[0], storage)
	if err != nil {
		t.Fatalf("PropagateUpward failed: %v", err)
	}

	// Should generate at least one parent tile at Z=19
	if len(generated) == 0 {
		t.Error("Expected at least one parent tile to be generated")
	}

	// First generated should be at Z=19
	if generated[0].Z != 19 {
		t.Errorf("Expected first generated tile at Z=19, got Z=%d", generated[0].Z)
	}

	t.Logf("PropagateUpward generated %d parent tiles", len(generated))
	for _, c := range generated {
		t.Logf("  Generated: %s", c.String())
	}
}
