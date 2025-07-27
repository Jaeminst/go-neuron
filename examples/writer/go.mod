module github.com/Jaeminst/go-neuron/examples/writer

go 1.24.2

replace (
	github.com/Jaeminst/go-neuron => ../..
	github.com/Jaeminst/go-neuron/examples/common => ../common
)

require (
	github.com/Jaeminst/go-neuron v0.1.0
	github.com/Jaeminst/go-neuron/examples/common v0.1.0
)

require (
	github.com/edsrzf/mmap-go v1.2.0 // indirect
	github.com/gofrs/flock v0.12.1 // indirect
	golang.org/x/sys v0.26.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
