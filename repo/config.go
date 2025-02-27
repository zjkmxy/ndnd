package repo

import (
	"fmt"
	"os"
	"path/filepath"
)

type Config struct {
	// StorageDir is the directory to store data.
	StorageDir string `json:"storage_dir"`
}

func (c *Config) Parse() error {
	if c.StorageDir == "" {
		return fmt.Errorf("storage-dir must be set")
	} else {
		path, err := filepath.Abs(c.StorageDir)
		if err != nil {
			return fmt.Errorf("failed to get absolute path: %w", err)
		}
		if err := os.MkdirAll(path, 0755); err != nil {
			return fmt.Errorf("failed to create storage directory: %w", err)
		}
		c.StorageDir = path
	}
	return nil
}

func DefaultConfig() *Config {
	return &Config{
		StorageDir: "", // invalid
	}
}
