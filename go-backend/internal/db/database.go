package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

// Database handles all database operations
type Database struct {
	db *sql.DB
}

// TileRecord represents a tile entry in the database
type TileRecord struct {
	ID         int64
	Z          int
	X          int
	Y          int
	UploadTime time.Time
	LocalPath  string
	IsMerged   bool
}

// NewDatabase creates or opens the SQLite database
func NewDatabase(dbPath string) (*Database, error) {
	// Create parent directory if it doesn't exist
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	// Open database
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	database := &Database{db: db}

	// Initialize schema
	if err := database.initSchema(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return database, nil
}

// initSchema creates the database schema if it doesn't exist
func (d *Database) initSchema() error {
	schema := `
	CREATE TABLE IF NOT EXISTS tiles (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		z INTEGER NOT NULL,
		x INTEGER NOT NULL,
		y INTEGER NOT NULL,
		upload_time DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		local_path TEXT NOT NULL,
		is_merged BOOLEAN NOT NULL DEFAULT 0,
		UNIQUE(z, x, y)
	);
	
	CREATE INDEX IF NOT EXISTS idx_coordinates ON tiles(z, x, y);
	CREATE INDEX IF NOT EXISTS idx_merge_status ON tiles(is_merged);
	CREATE INDEX IF NOT EXISTS idx_upload_time ON tiles(upload_time);
	`

	_, err := d.db.Exec(schema)
	if err != nil {
		return fmt.Errorf("failed to create schema: %w", err)
	}

	return nil
}

// InsertTile adds a new tile record to the database
func (d *Database) InsertTile(z, x, y int, localPath string, isMerged bool) (int64, error) {
	query := `
		INSERT INTO tiles (z, x, y, local_path, is_merged)
		VALUES (?, ?, ?, ?, ?)
	`

	result, err := d.db.Exec(query, z, x, y, localPath, isMerged)
	if err != nil {
		return 0, fmt.Errorf("failed to insert tile: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("failed to get last insert ID: %w", err)
	}

	return id, nil
}

// GetTile retrieves a tile record by coordinates
func (d *Database) GetTile(z, x, y int) (*TileRecord, error) {
	query := `
		SELECT id, z, x, y, upload_time, local_path, is_merged
		FROM tiles
		WHERE z = ? AND x = ? AND y = ?
	`

	var record TileRecord
	err := d.db.QueryRow(query, z, x, y).Scan(
		&record.ID,
		&record.Z,
		&record.X,
		&record.Y,
		&record.UploadTime,
		&record.LocalPath,
		&record.IsMerged,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get tile: %w", err)
	}

	return &record, nil
}

// TileExists checks if a tile exists at the given coordinates
func (d *Database) TileExists(z, x, y int) (bool, error) {
	query := `SELECT COUNT(*) FROM tiles WHERE z = ? AND x = ? AND y = ?`

	var count int
	err := d.db.QueryRow(query, z, x, y).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check tile existence: %w", err)
	}

	return count > 0, nil
}

// GetAllTiles returns all tile records (for recovery/debugging)
func (d *Database) GetAllTiles() ([]TileRecord, error) {
	query := `SELECT id, z, x, y, upload_time, local_path, is_merged FROM tiles ORDER BY upload_time DESC`

	rows, err := d.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query tiles: %w", err)
	}
	defer rows.Close()

	var tiles []TileRecord
	for rows.Next() {
		var record TileRecord
		err := rows.Scan(
			&record.ID,
			&record.Z,
			&record.X,
			&record.Y,
			&record.UploadTime,
			&record.LocalPath,
			&record.IsMerged,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan tile: %w", err)
		}
		tiles = append(tiles, record)
	}

	return tiles, nil
}

// Close closes the database connection
func (d *Database) Close() error {
	return d.db.Close()
}

// GetStats returns database statistics
func (d *Database) GetStats() (totalTiles int, mergedTiles int, err error) {
	// Get total tiles
	err = d.db.QueryRow(`SELECT COUNT(*) FROM tiles`).Scan(&totalTiles)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to count total tiles: %w", err)
	}

	// Get merged tiles
	err = d.db.QueryRow(`SELECT COUNT(*) FROM tiles WHERE is_merged = 1`).Scan(&mergedTiles)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to count merged tiles: %w", err)
	}

	return totalTiles, mergedTiles, nil
}

// GetLeafTileCount returns the number of original (non-merged) tiles at a specific zoom level
func (d *Database) GetLeafTileCount(z int) (int, error) {
	query := `SELECT COUNT(*) FROM tiles WHERE z = ? AND is_merged = 0`
	var count int
	err := d.db.QueryRow(query, z).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count leaf tiles: %w", err)
	}
	return count, nil
}
