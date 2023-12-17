package config

import (
	"errors"
	"fmt"
	"log"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Username    string `yaml:"username"`
	Password    string `yaml:"password"`
	APIEndpoint string `yaml:"api_endpoint"`

	Machines map[string]Machine `json:"machines"`
}

func GetConfig(path string) (Config, error) {
	var config Config

	log.Printf("Reading config file %s", path)

	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		return config, fmt.Errorf("File %s does not exist: %v", path, err)
	}

	b, err := os.ReadFile(path)
	if err != nil {
		return config, err
	}
	err = yaml.Unmarshal(b, &config)
	if err != nil {
		return config, err
	}

	return config, nil
}
