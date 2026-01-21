package db

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestDatabaseOperations(t *testing.T) {
	// Create temporary database
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := NewDatabase(dbPath)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	// Test InsertTile
	id, err := db.InsertTile(20, 100, 100, "/storage/tiles/20/100/100.webp", false)
	if err != nil {
		t.Fatalf("Failed to insert tile: %v", err)
	}
	if id <= 0 {
		t.Errorf("Expected positive ID, got %d", id)
	}

	// Test TileExists
	exists, err := db.TileExists(20, 100, 100)
	if err != nil {
		t.Fatalf("Failed to check tile existence: %v", err)
	}
	if !exists {
		t.Error("Tile should exist")
	}

	// Test GetTile
	tile, err := db.GetTile(20, 100, 100)
	if err != nil {
		t.Fatalf("Failed to get tile: %v", err)
	}
	if tile == nil {
		t.Fatal("Expected tile record, got nil")
	}
	if tile.Z != 20 || tile.X != 100 || tile.Y != 100 {
		t.Errorf("Wrong coordinates: Z:%d X:%d Y:%d", tile.Z, tile.X, tile.Y)
	}

	// Test GetStats
	total, merged, err := db.GetStats()
	if err != nil {
		t.Fatalf("Failed to get stats: %v", err)
	}
	if total != 1 {
		t.Errorf("Expected 1 total tile, got %d", total)
	}
	if merged != 0 {
		t.Errorf("Expected 0 merged tiles, got %d", merged)
	}

	t.Logf("Database operations test passed: inserted tile with ID %d", id)
}

func TestRecovery(t *testing.T) {
	// Create temporary directory structure
	tmpDir := t.TempDir()
	storageRoot := filepath.Join(tmpDir, "storage")

	// Create some fake tile files
	testTiles := []struct {
		z, x, y int
	}{
		{20, 0, 0},
		{20, 1, 0},
		{20, 0, 1},
		{19, 0, 0},
	}

	for _, tile := range testTiles {
		tilePath := filepath.Join(storageRoot, filepath.Join(
			filepath.Join(fmt.Sprintf("%d", tile.z), fmt.Sprintf("%d", tile.x)),
			fmt.Sprintf("%d.webp", tile.y),
		))
		dir := filepath.Dir(tilePath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}
		if err := os.WriteFile(tilePath, []byte("fake image"), 0644); err != nil {
			t.Fatalf("Failed to create fake tile: %v", err)
		}
	}

	// Create database
	dbPath := filepath.Join(tmpDir, "recovery.db")
	db, err := NewDatabase(dbPath)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	// Run recovery
	recovery := NewRecovery(db, storageRoot)
	count, err := recovery.RebuildFromFileSystem()
	if err != nil {
		t.Fatalf("Recovery failed: %v", err)
	}

	if count != len(testTiles) {
		t.Errorf("Expected to recover %d tiles, got %d", len(testTiles), count)
	}

	// Verify all tiles are in database
	for _, tile := range testTiles {
		exists, err := db.TileExists(tile.z, tile.x, tile.y)
		if err != nil {
			t.Fatalf("Failed to check tile existence: %v", err)
		}
		if !exists {
			t.Errorf("Tile (%d,%d,%d) should exist after recovery", tile.z, tile.x, tile.y)
		}
	}

	t.Log("Recovery test passed")
}
