package common

import (
	"os"

	"gopkg.in/yaml.v3"
)

type AppConfig struct {
	Message string
	Count   int
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
