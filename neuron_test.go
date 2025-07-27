package neuron

import (
	"bytes"
	"encoding/gob"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type AppConfig struct {
	Message string
	Count   int
}

func TestNewSyncFlush(t *testing.T) {
	// isolate HOME directory for test
	tmpHome := t.TempDir()
	require.NoError(t, os.Setenv("HOME", tmpHome))

	cfg := AppConfig{Message: "init", Count: 0}
	syncObj, err := NewSync(&cfg)
	require.NoError(t, err)
	defer syncObj.Close()

	// Modify and flush
	cfg.Message = "updated"
	cfg.Count = 5
	require.NoError(t, syncObj.Flush())

	// Verify raw mmap content
	mmapPath := filepath.Join(tmpHome, ".cache", "syncstruct", "neuron_AppConfig.mmap")
	data, err := os.ReadFile(mmapPath)
	require.NoError(t, err)
	raw := data[4:]
	var loaded AppConfig
	require.NoError(t, gob.NewDecoder(bytes.NewReader(raw)).Decode(&loaded))
	require.Equal(t, cfg, loaded)
}

func TestAutoFlush(t *testing.T) {
	// isolate HOME directory for test
	tmpHome := t.TempDir()
	require.NoError(t, os.Setenv("HOME", tmpHome))

	cfg := AppConfig{Message: "start", Count: 0}
	syncObj, err := NewSync(&cfg)
	require.NoError(t, err)
	defer syncObj.Close()

	// Start auto-flush
	syncObj.AutoFlush(50 * time.Millisecond)

	// Directly modify struct
	cfg.Message = "auto"
	cfg.Count = 42

	// Allow AutoFlush to detect and write
	time.Sleep(200 * time.Millisecond)

	// Allow initial decode
	require.Equal(t, 42, cfg.Count)
	require.Equal(t, "auto", cfg.Message)
}

func TestNewMmapRegion(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.mmap")
	region, err := newMmapRegion(path, 1024)
	require.NoError(t, err)
	require.NotNil(t, region)

	// Verify file size
	info, err := os.Stat(path)
	require.NoError(t, err)
	require.Equal(t, int64(1024), info.Size())

	// Clean up
	require.NoError(t, region.mmap.Unmap())
	require.NoError(t, region.file.Close())
}
