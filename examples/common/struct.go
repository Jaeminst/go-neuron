package common

import (
	"os"

	"gopkg.in/yaml.v3"
)

type XYZ struct {
	X float64
	Y float64
	Z float64
}

type AppConfig struct {
	Message string
	Count   int
	XYZ     *XYZ
}

func NewConfig[T any](path string) (T, error) {
	var cfg T
	f, err := os.Open(path)
	if err != nil {
		return cfg, err
	}
	defer f.Close()
	dec := yaml.NewDecoder(f)
	err = dec.Decode(&cfg)
	return cfg, err
}
