package sidecar

import (
	"io/ioutil"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/yaml"
)

type EnvironmentVariable struct {
	Name       string `yaml:"name"`
	Container  string `yaml:"container"`
	Annotation string `yaml:"annotation"`
}

type Config struct {
	Containers           []corev1.Container    `yaml:"containers"`
	Volumes              []corev1.Volume       `yaml:"volumes"`
	EnvironmentVariables []EnvironmentVariable `yaml:"environmentVariables"`
}

func LoadConfig(file string) (*Config, error) {
	configContent, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, err
	}

	cfg := &Config{}
	if err := yaml.Unmarshal(configContent, cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}
