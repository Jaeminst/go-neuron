# go-neuron 🧠

> Real-time shared struct synchronization across processes in Go — like neurons firing between brains.

`go-neuron` is a Go library that allows you to safely share, observe, and update struct values across multiple processes in real-time.  
It uses memory-mapped files (`mmap`) under the hood, with built-in gob serialization and atomic versioning.

---

## 🚀 Features

- 🧠 Real-time cross-process struct sync
- 🔁 Background update propagation (no polling needed)
- 🧩 Type-safe generic API (`Sync[T]`)
- 🔐 Lock-safe updates
- 📦 No external dependencies (aside from `mmap` & `flock` & `fsnotify`)

---

## 💡 Use Case

```go
type Status struct {
    LoadAverage float64
    ActiveUsers int
    AlertsOn    bool
    Message     string
}

// Create a shared memory binding
status := Status{}
live, err := neuron.NewSync(&status)
if err != nil {
    log.Fatal(err)
}

// React to changes automatically
live.OnChange(func(s Status) {
    fmt.Printf("Updated: %+v\n", s)
})
```

### In a separate process

```go
status := Status{}
if _, err := neuron.NewSync(&status); err != nil {
    log.Fatal(err)
}
status.LoadAverage = 70.0
status.AlertsOn = true
```

## 📦 Installation

```sh
go get github.com/Jaeminst/go-neuron
```

## 📁 Roadmap

- Basic gob-encoded sync via mmap
- OnChange callback for live updates
- JSON serialization support
- Shared memory fallback for embedded devices
- CLI for introspection: neuronctl
