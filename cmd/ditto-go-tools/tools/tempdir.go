// Copyright 2025 DittoLive Incorporated. All rights reserved.

package tools

import (
	"crypto/rand"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// TempPersistenceDir creates a temporary directory suitable for use as Ditto persistence
// directory.
type TempPersistenceDir struct {
	autoCleanup bool
	path        string
}

// NewTempPersistenceDir creates a temporary directory.
//
// If autoCleanup is true, then the TempPersistenceDir's
// Close method will delete the directory it created. It is the caller's
// responsibility to ensure that any Ditto instance using that directory
// will be shut down before that happens.
func NewTempPersistenceDir(autoCleanup bool) (*TempPersistenceDir, error) {
	// Generate a random number for the directory name
	randBytes := make([]byte, 8)
	if _, err := rand.Read(randBytes); err != nil {
		return nil, fmt.Errorf("failed to generate random bytes: %w", err)
	}
	randNum := fmt.Sprintf("%x", randBytes)

	var dittoDir string
	if runtime.GOOS == "windows" {
		// On Windows, use LocalAppData
		localAppData := os.Getenv("LOCALAPPDATA")
		if localAppData == "" {
			return nil, fmt.Errorf("LOCALAPPDATA environment variable not set")
		}
		dittoDir = filepath.Join(localAppData, "Ditto", fmt.Sprintf("ditto-go-tools-%s", randNum))
	} else {
		// On Unix-like systems, use temp directory
		tempDir := os.TempDir()
		dittoDir = filepath.Join(tempDir, "ditto", fmt.Sprintf("ditto-go-tools-%s", randNum))
	}

	// Clear existing contents if present
	if err := os.RemoveAll(dittoDir); err != nil && !os.IsNotExist(err) {
		// Log but don't fail - we'll fail later if we can't create the directory
		fmt.Fprintf(os.Stderr, "Warning: failed to clean existing directory %s: %v\n", dittoDir, err)
	}

	return &TempPersistenceDir{
		autoCleanup: autoCleanup,
		path:        dittoDir,
	}, nil
}

// Path returns the path to the directory, which can be passed into the Ditto constructor.
func (t *TempPersistenceDir) Path() string {
	return t.path
}

// RemoveAll removes the created directory.
func (t *TempPersistenceDir) RemoveAll() error {
	if t.path == "" {
		return fmt.Errorf("directory path is empty")
	}
	return os.RemoveAll(t.path)
}

// Close removes the created directory if autoCleanup was set to true.
func (t *TempPersistenceDir) Close() error {
	if t.autoCleanup && t.path != "" {
		return t.RemoveAll()
	}
	return nil
}
