package config

import (
	"os"
	"path/filepath"
)

// Config holds all application configuration
type Config struct {
	// Server configuration
	ServerPort string

	// Storage paths
	StorageRoot  string
	TilesDir     string
	DatabasePath string

	// Image processing settings
	TileSize        int
	WebPQuality     int
	MaxUploadSizeMB int

	// Perlin noise settings for AI protection
	PerlinFrequency float64
	PerlinOctaves   int32
	PerlinOpacity   float64
	PerlinAmplitude float64
}

// LoadConfig loads configuration from environment variables with sensible defaults
func LoadConfig() *Config {
	storageRoot := getEnv("STORAGE_ROOT", "./storage")

	return &Config{
		ServerPort:      getEnv("PORT", "8080"),
		StorageRoot:     storageRoot,
		TilesDir:        filepath.Join(storageRoot, "tiles"),
		DatabasePath:    filepath.Join(storageRoot, "galaxy.db"),
		TileSize:        256,
		WebPQuality:     60,
		MaxUploadSizeMB: 10,

		// Perlin noise parameters for AI protection
		PerlinFrequency: 8.0,
		PerlinOctaves:   4,
		PerlinOpacity:   0.6,  // 60% opacity
		PerlinAmplitude: 30.0, // ±30 RGB value variance
	}
}

// getEnv retrieves environment variable or returns default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
