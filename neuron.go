package neuron

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/gob"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"time"

	"github.com/edsrzf/mmap-go"
	"github.com/fsnotify/fsnotify"
	"github.com/gofrs/flock"
)

var gobOnce sync.Once

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
	gobOnce.Do(func() {
		gob.Register(*ptr)
	})

	home, _ := os.UserHomeDir()
	typeName := fmt.Sprintf("%T", *ptr)
	path := filepath.Join(home, ".cache/go-neuron", typeName+".mmap")
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
	if err := s.watch(path); err != nil {
		return nil, err
	}
	s.autoFlush(100 * time.Millisecond)
	return s, nil
}

func (s *Sync[T]) Close() {
	close(s.stopCh)
	s.mem.close()
}

func (s *Sync[T]) Flush() error {
	if err := s.mem.lock.Lock(); err != nil {
		return err
	}
	defer s.mem.lock.Unlock()

	data, err := s.encode()
	if err != nil {
		return err
	}
	return s.flush(data)
}

func (s *Sync[T]) OnChange(cb func(T)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.callbacks = append(s.callbacks, cb)
}

func (s *Sync[T]) watch(path string) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}

	if err := watcher.Add(path); err != nil {
		defer watcher.Close()
		return err
	}

	go func() {
		defer watcher.Close()
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if event.Name == path && event.Op&fsnotify.Write != 0 {
					memVer := binary.LittleEndian.Uint32(s.mem.mmap[0:4])
					if memVer != s.version {
						s.version = memVer
						raw := s.mem.mmap[4:]
						var tmp T
						if err := gob.NewDecoder(bytes.NewReader(raw)).Decode(&tmp); err == nil {
							s.mu.Lock()
							*s.ptr = tmp
							for _, cb := range s.callbacks {
								go func(cb func(T), v T) {
									defer func() {
										if r := recover(); r != nil {
											fmt.Printf("callback panic: %v\n", r)
										}
									}()
									cb(v)
								}(cb, tmp)
							}
							s.mu.Unlock()
						} else {
							fmt.Printf("decode error: %v\n", err)
						}
					}
				}
			case err := <-watcher.Errors:
				fmt.Println("watch error:", err)
			case <-s.stopCh:
				return
			}
		}
	}()

	return nil
}

func (s *Sync[T]) encode() (data []byte, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(*s.ptr); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (s *Sync[T]) encodeAndHash() (data []byte, hash []byte, err error) {
	raw, err := s.encode()
	if err != nil {
		return nil, nil, err
	}
	h := sha256.Sum256(raw)
	return raw, h[:], nil
}

func (s *Sync[T]) flush(data []byte) error {
	if len(data)+4 > len(s.mem.mmap) {
		return fmt.Errorf("data too large for mmap: required %d, available %d", len(data)+4, len(s.mem.mmap))
	}

	copy(s.mem.mmap[4:], data)
	nextVersion := s.version + 1
	binary.LittleEndian.PutUint32(s.mem.mmap[0:4], nextVersion)
	if err := s.mem.mmap.Flush(); err != nil {
		return err
	}
	s.version = nextVersion
	return nil
}

func (s *Sync[T]) flushWithData(data []byte) error {
	if err := s.mem.lock.Lock(); err != nil {
		return err
	}
	defer s.mem.lock.Unlock()

	return s.flush(data)
}

func (s *Sync[T]) autoFlush(interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		var lastHash []byte
		for {
			select {
			case <-ticker.C:
				data, currentHash, err := s.encodeAndHash()
				if err != nil {
					fmt.Printf("encode and hash error: %v\n", err)
					continue
				}

				if !bytes.Equal(lastHash, currentHash) {
					if err := s.flushWithData(data); err != nil {
						fmt.Printf("auto flush error: %v\n", err)
					} else {
						lastHash = currentHash
					}
				}
			case <-s.stopCh:
				return
			}
		}
	}()
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
