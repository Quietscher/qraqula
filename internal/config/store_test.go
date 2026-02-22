package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadMissingFile(t *testing.T) {
	dir := t.TempDir()
	s := NewStore(dir)
	if err := s.Load(); err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if s.Config.ActiveEnv != "" {
		t.Errorf("expected empty ActiveEnv, got %q", s.Config.ActiveEnv)
	}
	if len(s.Config.Environments) != 0 {
		t.Errorf("expected no environments, got %d", len(s.Config.Environments))
	}
	if len(s.Config.GlobalHeaders) != 0 {
		t.Errorf("expected no global headers, got %d", len(s.Config.GlobalHeaders))
	}
}

func TestSaveLoadRoundTrip(t *testing.T) {
	dir := t.TempDir()
	s := NewStore(dir)
	s.Config = Config{
		ActiveEnv: "dev",
		Environments: []Environment{
			{
				Name:     "dev",
				Endpoint: "https://dev.api.com/graphql",
				Headers:  []Header{{Key: "Authorization", Value: "Bearer xxx", Enabled: true}},
			},
			{
				Name:     "prod",
				Endpoint: "https://api.com/graphql",
			},
		},
		GlobalHeaders: []Header{
			{Key: "Accept", Value: "application/json", Enabled: true},
			{Key: "X-Debug", Value: "true", Enabled: false},
		},
	}
	if err := s.Save(); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	s2 := NewStore(dir)
	if err := s2.Load(); err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if s2.Config.ActiveEnv != "dev" {
		t.Errorf("expected ActiveEnv=dev, got %q", s2.Config.ActiveEnv)
	}
	if len(s2.Config.Environments) != 2 {
		t.Fatalf("expected 2 envs, got %d", len(s2.Config.Environments))
	}
	if s2.Config.Environments[0].Endpoint != "https://dev.api.com/graphql" {
		t.Errorf("wrong endpoint: %s", s2.Config.Environments[0].Endpoint)
	}
	if len(s2.Config.Environments[0].Headers) != 1 {
		t.Errorf("expected 1 env header, got %d", len(s2.Config.Environments[0].Headers))
	}
	if len(s2.Config.GlobalHeaders) != 2 {
		t.Fatalf("expected 2 global headers, got %d", len(s2.Config.GlobalHeaders))
	}
	if !s2.Config.GlobalHeaders[0].Enabled {
		t.Error("expected first global header enabled")
	}
	if s2.Config.GlobalHeaders[1].Enabled {
		t.Error("expected second global header disabled")
	}
}

func TestSaveAtomicCreatesFile(t *testing.T) {
	dir := t.TempDir()
	s := NewStore(dir)
	if err := s.Save(); err != nil {
		t.Fatalf("Save() error: %v", err)
	}
	path := filepath.Join(dir, configFile)
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("config file not created: %v", err)
	}
	// .tmp file should not remain
	if _, err := os.Stat(path + ".tmp"); !os.IsNotExist(err) {
		t.Error("temp file should not remain after save")
	}
}

func TestMergedHeadersGlobalOnly(t *testing.T) {
	c := Config{
		GlobalHeaders: []Header{
			{Key: "Accept", Value: "application/json", Enabled: true},
			{Key: "X-Debug", Value: "true", Enabled: false},
		},
	}
	h := c.MergedHeaders()
	if h["Accept"] != "application/json" {
		t.Errorf("expected Accept header, got %v", h)
	}
	if _, ok := h["X-Debug"]; ok {
		t.Error("disabled header should not appear")
	}
}

func TestMergedHeadersEnvOnly(t *testing.T) {
	c := Config{
		ActiveEnv: "dev",
		Environments: []Environment{
			{
				Name: "dev",
				Headers: []Header{
					{Key: "Authorization", Value: "Bearer token", Enabled: true},
				},
			},
		},
	}
	h := c.MergedHeaders()
	if h["Authorization"] != "Bearer token" {
		t.Errorf("expected Authorization header, got %v", h)
	}
}

func TestMergedHeadersEnvOverridesGlobal(t *testing.T) {
	c := Config{
		ActiveEnv: "dev",
		Environments: []Environment{
			{
				Name: "dev",
				Headers: []Header{
					{Key: "Authorization", Value: "env-token", Enabled: true},
				},
			},
		},
		GlobalHeaders: []Header{
			{Key: "Authorization", Value: "global-token", Enabled: true},
			{Key: "Accept", Value: "application/json", Enabled: true},
		},
	}
	h := c.MergedHeaders()
	if h["Authorization"] != "env-token" {
		t.Errorf("env should override global, got %q", h["Authorization"])
	}
	if h["Accept"] != "application/json" {
		t.Errorf("global Accept should be preserved, got %v", h)
	}
}

func TestMergedHeadersNoActiveEnv(t *testing.T) {
	c := Config{
		ActiveEnv: "",
		Environments: []Environment{
			{Name: "dev", Headers: []Header{{Key: "X-Env", Value: "dev", Enabled: true}}},
		},
		GlobalHeaders: []Header{
			{Key: "Accept", Value: "application/json", Enabled: true},
		},
	}
	h := c.MergedHeaders()
	if _, ok := h["X-Env"]; ok {
		t.Error("no active env means no env headers")
	}
	if h["Accept"] != "application/json" {
		t.Error("global headers should still be present")
	}
}

func TestActiveEnvironmentFound(t *testing.T) {
	c := Config{
		ActiveEnv: "staging",
		Environments: []Environment{
			{Name: "dev"},
			{Name: "staging", Endpoint: "https://staging.api.com"},
			{Name: "prod"},
		},
	}
	env := c.ActiveEnvironment()
	if env == nil {
		t.Fatal("expected non-nil environment")
	}
	if env.Name != "staging" {
		t.Errorf("expected staging, got %s", env.Name)
	}
	if env.Endpoint != "https://staging.api.com" {
		t.Errorf("wrong endpoint: %s", env.Endpoint)
	}
}

func TestActiveEnvironmentNotFound(t *testing.T) {
	c := Config{
		ActiveEnv:    "missing",
		Environments: []Environment{{Name: "dev"}},
	}
	if c.ActiveEnvironment() != nil {
		t.Error("expected nil for missing environment")
	}
}

func TestActiveEnvironmentEmpty(t *testing.T) {
	c := Config{
		ActiveEnv:    "",
		Environments: []Environment{{Name: "dev"}},
	}
	if c.ActiveEnvironment() != nil {
		t.Error("expected nil for empty ActiveEnv")
	}
}

func TestEnvNames(t *testing.T) {
	c := Config{
		Environments: []Environment{
			{Name: "dev"},
			{Name: "staging"},
			{Name: "prod"},
		},
	}
	names := c.EnvNames()
	if len(names) != 3 {
		t.Fatalf("expected 3 names, got %d", len(names))
	}
	if names[0] != "dev" || names[1] != "staging" || names[2] != "prod" {
		t.Errorf("unexpected names: %v", names)
	}
}

func TestEnvNamesEmpty(t *testing.T) {
	c := Config{}
	names := c.EnvNames()
	if len(names) != 0 {
		t.Errorf("expected empty names, got %v", names)
	}
}
