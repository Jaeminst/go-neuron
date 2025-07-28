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
	mmapPath := filepath.Join(tmpHome, ".cache", "go-neuron", "neuron.AppConfig.mmap")
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

func TestOnChangeCallback(t *testing.T) {
	tmpHome := t.TempDir()
	require.NoError(t, os.Setenv("HOME", tmpHome))

	cfg1 := AppConfig{Message: "original", Count: 1}
	syncObj1, err := NewSync(&cfg1)
	require.NoError(t, err)
	defer syncObj1.Close()

	cfg2 := AppConfig{}
	syncObj2, err := NewSync(&cfg2)
	require.NoError(t, err)
	defer syncObj2.Close()

	callbackCalled := make(chan AppConfig, 1)
	syncObj2.OnChange(func(newVal AppConfig) {
		callbackCalled <- newVal
	})

	// 변경 및 flush → 다른 인스턴스가 감지
	cfg1.Message = "from test"
	cfg1.Count = 99
	require.NoError(t, syncObj1.Flush())

	select {
	case got := <-callbackCalled:
		require.Equal(t, "from test", got.Message)
		require.Equal(t, 99, got.Count)
	case <-time.After(1 * time.Second):
		t.Fatal("OnChange callback not triggered within timeout")
	}
}
