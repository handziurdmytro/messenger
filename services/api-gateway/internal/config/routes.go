package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type RouteConfig struct {
	ID           string `yaml:"id"`
	Prefix       string `yaml:"prefix"`
	Target       string `yaml:"target"`
	StripPrefix  bool   `yaml:"strip_prefix"`
	AuthRequired bool   `yaml:"auth_required"`
}

type routesFile struct {
	Routes []RouteConfig `yaml:"routes"`
}

func loadRoutes(path string) ([]RouteConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read routes file %q: %w", path, err)
	}

	var f routesFile
	if err := yaml.Unmarshal(data, &f); err != nil {
		return nil, fmt.Errorf("parse routes file: %w", err)
	}

	if err := validateRoutes(f.Routes); err != nil {
		return nil, err
	}

	return f.Routes, nil
}

func validateRoutes(routes []RouteConfig) error {
	if len(routes) == 0 {
		return fmt.Errorf("routes file is empty")
	}

	valid := make(map[string]struct{}, len(routes))
	for i, r := range routes {
		if r.ID == "" {
			return fmt.Errorf("route[%d]: id is required", i)
		}
		if r.Prefix == "" {
			return fmt.Errorf("route %q: prefix is required", r.ID)
		}
		if r.Target == "" {
			return fmt.Errorf("route %q: target is required", r.ID)
		}
		if _, dup := valid[r.ID]; dup {
			return fmt.Errorf("route %q: duplicate id", r.ID)
		}
		valid[r.ID] = struct{}{}
	}

	return nil
}
