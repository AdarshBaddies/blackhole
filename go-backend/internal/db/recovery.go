package db

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Recovery handles database recovery from filesystem
type Recovery struct {
	db          *Database
	storageRoot string
}

// NewRecovery creates a new recovery instance
func NewRecovery(db *Database, storageRoot string) *Recovery {
	return &Recovery{
		db:          db,
		storageRoot: storageRoot,
	}
}

// RebuildFromFileSystem scans the storage directory and rebuilds the database
func (r *Recovery) RebuildFromFileSystem() (int, error) {
	count := 0

	err := filepath.Walk(r.storageRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Parse coordinates from path
		// Expected format: {storageRoot}/{z}/{x}/{y}.webp
		relPath, err := filepath.Rel(r.storageRoot, path)
		if err != nil {
			return nil // Skip files we can't parse
		}

		var z, x, y int
		ext := filepath.Ext(relPath)
		// Remove extension
		relPath = relPath[:len(relPath)-len(ext)]

		// Replace backslashes with forward slashes for consistent parsing
		relPath = strings.ReplaceAll(relPath, string(os.PathSeparator), "/")

		_, err = fmt.Sscanf(relPath, "%d/%d/%d", &z, &x, &y)
		if err != nil {
			// Skip files that don't match pattern
			return nil
		}

		// Check if tile already exists in database
		exists, err := r.db.TileExists(z, x, y)
		if err != nil {
			return fmt.Errorf("failed to check tile existence for (%d,%d,%d): %w", z, x, y, err)
		}

		if !exists {
			// Insert tile into database
			_, err = r.db.InsertTile(z, x, y, path, false)
			if err != nil {
				return fmt.Errorf("failed to insert tile (%d,%d,%d): %w", z, x, y, err)
			}
			count++
		}

		return nil
	})

	if err != nil {
		return count, fmt.Errorf("recovery failed: %w", err)
	}

	return count, nil
}

// ValidateConsistency checks that all database entries have corresponding files
func (r *Recovery) ValidateConsistency() (missing []string, err error) {
	tiles, err := r.db.GetAllTiles()
	if err != nil {
		return nil, fmt.Errorf("failed to get tiles: %w", err)
	}

	for _, tile := range tiles {
		if _, err := os.Stat(tile.LocalPath); os.IsNotExist(err) {
			missing = append(missing, fmt.Sprintf("Z:%d X:%d Y:%d (Path: %s)", tile.Z, tile.X, tile.Y, tile.LocalPath))
		}
	}

	return missing, nil
}
