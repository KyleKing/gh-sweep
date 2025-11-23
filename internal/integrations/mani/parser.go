package mani

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config represents a mani configuration
type Config struct {
	Projects []Project          `yaml:"projects"`
	Tasks    map[string]Task    `yaml:"tasks"`
}

// Project represents a mani project
type Project struct {
	Name string   `yaml:"name"`
	Path string   `yaml:"path"`
	URL  string   `yaml:"url"`
	Tags []string `yaml:"tags"`
}

// Task represents a mani task
type Task struct {
	Cmd  string `yaml:"cmd"`
	Desc string `yaml:"desc"`
}

// Parse parses a mani.yaml file
func Parse(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read mani config: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse mani config: %w", err)
	}

	return &config, nil
}

// ListProjects returns all projects from the config
func (c *Config) ListProjects() []Project {
	return c.Projects
}

// GetTask returns a task by name
func (c *Config) GetTask(name string) (Task, bool) {
	task, ok := c.Tasks[name]
	return task, ok
}

// FilterProjectsByTag filters projects by tag
func (c *Config) FilterProjectsByTag(tag string) []Project {
	filtered := []Project{}

	for _, p := range c.Projects {
		for _, t := range p.Tags {
			if t == tag {
				filtered = append(filtered, p)
				break
			}
		}
	}

	return filtered
}
