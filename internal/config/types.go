package config

// Header is a single key-value pair with an Enabled toggle.
type Header struct {
	Key     string `json:"key"`
	Value   string `json:"value"`
	Enabled bool   `json:"enabled"`
}

// Environment represents a named environment (dev, staging, prod, etc.).
type Environment struct {
	Name      string   `json:"name"`
	Endpoint  string   `json:"endpoint"`
	Headers   []Header `json:"headers"`
	Variables string   `json:"variables"`
}

// Config is the top-level configuration persisted to disk.
type Config struct {
	ActiveEnv     string        `json:"activeEnv"`
	Environments  []Environment `json:"environments"`
	GlobalHeaders []Header      `json:"globalHeaders"`
}

// MergedHeaders returns global + active environment headers merged.
// Environment headers override global headers with the same key.
// Only enabled headers are included.
func (c *Config) MergedHeaders() map[string]string {
	result := make(map[string]string)
	for _, h := range c.GlobalHeaders {
		if h.Enabled {
			result[h.Key] = h.Value
		}
	}
	if env := c.ActiveEnvironment(); env != nil {
		for _, h := range env.Headers {
			if h.Enabled {
				result[h.Key] = h.Value
			}
		}
	}
	return result
}

// ActiveEnvironment returns the active environment, or nil if none selected.
func (c *Config) ActiveEnvironment() *Environment {
	if c.ActiveEnv == "" {
		return nil
	}
	for i := range c.Environments {
		if c.Environments[i].Name == c.ActiveEnv {
			return &c.Environments[i]
		}
	}
	return nil
}

// EnvNames returns the list of environment names.
func (c *Config) EnvNames() []string {
	names := make([]string, len(c.Environments))
	for i, e := range c.Environments {
		names[i] = e.Name
	}
	return names
}
