package sidecar

import (
	"os"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"
)

type InjectorData struct {
	Selector       metav1.LabelSelector `yaml:"selector"`
	Containers     []string             `yaml:"container"`
	InitContainers []string             `yaml:"initContainers"`
	Volumes        []string             `yaml:"volumes"`
}

type EnvironmentVariable struct {
	Name       string `yaml:"name"`
	Container  string `yaml:"container"`
	Annotation string `yaml:"annotation"`
}

type Config struct {
	Injectors            []InjectorData        `yaml:"injectors"`
	Containers           []corev1.Container    `yaml:"containers"`
	Volumes              []corev1.Volume       `yaml:"volumes"`
	EnvironmentVariables []EnvironmentVariable `yaml:"environmentVariables"`
}

func LoadConfig(file string) (*Config, error) {
	configContent, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}

	cfg := &Config{}
	if err := yaml.Unmarshal(configContent, cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}
