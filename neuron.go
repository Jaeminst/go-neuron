package neuron

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/edsrzf/mmap-go"
	"github.com/gofrs/flock"
)

type Sync[T any] struct {
	mem       *mmapRegion
	ptr       *T
	version   uint32
	stopCh    chan struct{}
	mu        sync.Mutex
	callbacks []func(T)
}

func NewSync[T any](ptr *T) (*Sync[T], error) {
	// Validate ptr
	if ptr == nil {
		return nil, errors.New("NewSync: ptr must not be nil")
	}
	// Ensure T is a struct
	t := reflect.TypeOf(*ptr)
	if t.Kind() != reflect.Struct {
		return nil, fmt.Errorf("NewSync requires pointer to struct, got %T", ptr)
	}

	// Register type for gob
	gob.Register(*ptr)

	home, _ := os.UserHomeDir()
	typeName := fmt.Sprintf("%T", *ptr)
	safe := strings.ReplaceAll(typeName, ".", "_")
	path := filepath.Join(home, ".cache/syncstruct", safe+".mmap")
	_ = os.MkdirAll(filepath.Dir(path), 0755)

	size := 64 * 1024
	mem, err := newMmapRegion(path, size)
	if err != nil {
		return nil, err
	}

	memVer := binary.LittleEndian.Uint32(mem.mmap[0:4])
	if memVer != 0 {
		raw := mem.mmap[4:]
		_ = gob.NewDecoder(bytes.NewReader(raw)).Decode(ptr)
	}

	s := &Sync[T]{
		mem:     mem,
		ptr:     ptr,
		stopCh:  make(chan struct{}),
		version: memVer,
	}
	go s.syncLoop()
	return s, nil
}

func (s *Sync[T]) syncLoop() {
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			memVer := binary.LittleEndian.Uint32(s.mem.mmap[0:4])
			if memVer != s.version {
				s.version = memVer
				raw := s.mem.mmap[4:]
				if err := gob.NewDecoder(bytes.NewReader(raw)).Decode(s.ptr); err == nil {
					s.mu.Lock()
					for _, cb := range s.callbacks {
						go cb(*s.ptr)
					}
					s.mu.Unlock()
				}
			}
		case <-s.stopCh:
			return
		}
	}
}

func (s *Sync[T]) Flush() error {
	if err := s.mem.lock.Lock(); err != nil {
		return err
	}
	defer s.mem.lock.Unlock()

	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(*s.ptr); err != nil {
		return err
	}

	data := buf.Bytes()
	if len(data)+4 > len(s.mem.mmap) {
		return errors.New("data too large for mmap")
	}

	copy(s.mem.mmap[4:], data)
	s.version++
	binary.LittleEndian.PutUint32(s.mem.mmap[0:4], s.version)
	return s.mem.mmap.Flush()
}

func (s *Sync[T]) AutoFlush(interval time.Duration) {
	go func() {
		var last T = *s.ptr

		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				s.mu.Lock()
				changed := !reflect.DeepEqual(last, *s.ptr)
				s.mu.Unlock()

				if changed {
					if err := s.Flush(); err != nil {
						fmt.Printf("auto flush error: %v", err)
					} else {
						last = *s.ptr
					}
				}
			case <-s.stopCh:
				return
			}
		}
	}()
}

func (s *Sync[T]) OnChange(cb func(T)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.callbacks = append(s.callbacks, cb)
}

func (s *Sync[T]) Close() {
	close(s.stopCh)
	s.mem.close()
}

type mmapRegion struct {
	file *os.File
	mmap mmap.MMap
	lock *flock.Flock
}

func newMmapRegion(path string, size int) (*mmapRegion, error) {
	lock := flock.New(path + ".lock")
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return nil, err
	}
	if err := f.Truncate(int64(size)); err != nil {
		f.Close()
		return nil, err
	}
	m, err := mmap.Map(f, mmap.RDWR, 0)
	if err != nil {
		f.Close()
		return nil, err
	}
	return &mmapRegion{file: f, mmap: m, lock: lock}, nil
}

func (r *mmapRegion) close() {
	r.mmap.Unmap()
	r.file.Close()
}
