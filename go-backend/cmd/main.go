package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/AdarshBaddies/blackhole/go-backend/internal/api"
	"github.com/AdarshBaddies/blackhole/go-backend/internal/config"
	"github.com/AdarshBaddies/blackhole/go-backend/internal/db"
	"github.com/AdarshBaddies/blackhole/go-backend/internal/imageproc"
	"github.com/AdarshBaddies/blackhole/go-backend/internal/quadtree"
)

func main() {
	// Load configuration
	cfg := config.LoadConfig()

	log.Println("Starting Humanity Galaxy Backend...")
	log.Printf("Server Port: %s", cfg.ServerPort)
	log.Printf("Storage Root: %s", cfg.StorageRoot)
	log.Printf("Tiles Directory: %s", cfg.TilesDir)
	log.Printf("Database Path: %s", cfg.DatabasePath)

	// Initialize database
	database, err := db.NewDatabase(cfg.DatabasePath)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer database.Close()
	log.Println("✓ Database initialized")

	// Initialize image processor
	procConfig := &imageproc.ProcessorConfig{
		TileSize:        cfg.TileSize,
		WebPQuality:     cfg.WebPQuality,
		PerlinFrequency: cfg.PerlinFrequency,
		PerlinOctaves:   cfg.PerlinOctaves,
		PerlinOpacity:   cfg.PerlinOpacity,
		PerlinAmplitude: cfg.PerlinAmplitude,
	}
	processor := imageproc.NewProcessor(procConfig)
	log.Println("✓ Image processor initialized with Perlin noise protection")

	// Initialize storage
	storage := quadtree.NewFileSystemStorage(cfg.TilesDir)
	log.Println("✓ File system storage initialized")

	// Initialize quadtree manager
	qtManager := quadtree.NewQuadtreeManager(cfg.TilesDir, cfg.TileSize)
	log.Println("✓ Quadtree manager initialized")

	// Create API server
	server := api.NewServer(cfg, processor, database, storage, qtManager)

	// Setup graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Start server in goroutine
	go func() {
		addr := fmt.Sprintf(":%s", cfg.ServerPort)
		log.Printf("✓ Server listening on %s", addr)
		log.Printf("✓ Upload endpoint: POST http://localhost:%s/upload", cfg.ServerPort)
		log.Printf("✓ Tiles endpoint: GET http://localhost:%s/tiles/{z}/{x}/{y}.webp", cfg.ServerPort)

		if err := http.ListenAndServe(addr, server.Router()); err != nil {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	// Wait for shutdown signal
	<-sigChan
	log.Println("\nShutting down gracefully...")
	database.Close()
	log.Println("✓ Goodbye!")
}
