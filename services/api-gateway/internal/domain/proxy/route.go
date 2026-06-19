package proxy

import "strings"

type RouteID string

type Route struct {
	ID           RouteID
	Prefix       string
	Target       string
	StripPrefix  bool
	AuthRequired bool
}

func (r Route) Match(path string) bool {
	return strings.HasPrefix(path, r.Prefix)
}

func (r Route) RewritePath(originalPath string) string {
	if !r.StripPrefix {
		return originalPath
	}
	trimmed := strings.TrimPrefix(originalPath, r.Prefix)
	if trimmed == "" {
		return "/"
	}
	return trimmed
}
