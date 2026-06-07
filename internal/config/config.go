package config

import (
	"fmt"
	"os"

	"github.com/pelletier/go-toml/v2"
)

// Config holds runtime configuration from config.toml.
type Config struct {
	Plugins map[string]PluginConfig
}

// PluginConfig holds one [plugins.<name>] GTP plugin config.
type PluginConfig struct {
	Command      string
	Args         []string
	Cwd          string
	AutoStart    bool
	Capabilities []string
	Modules      []string
}

// Load reads a config.toml file and returns parsed config.
// Returns defaults if the file doesn't exist.
func Load(path string) *Config {
	cfg, _ := LoadStrict(path)
	return cfg
}

// LoadStrict reads a config.toml file and reports invalid TOML. Missing files
// still return the default config so projects can run without runtime config.
func LoadStrict(path string) (*Config, error) {
	cfg := &Config{}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return cfg, err
	}
	return Parse(string(data), path)
}

func Parse(src, name string) (*Config, error) {
	cfg := &Config{}
	var file configFile
	if err := toml.Unmarshal([]byte(src), &file); err != nil {
		return cfg, fmt.Errorf("invalid config %q: %w", name, err)
	}
	cfg.apply(file)
	return cfg, nil
}

type configFile struct {
	Plugins map[string]pluginSection `toml:"plugins"`
}

type pluginSection struct {
	Command      string   `toml:"command"`
	Args         []string `toml:"args"`
	Cwd          string   `toml:"cwd"`
	AutoStart    *bool    `toml:"autoStart"`
	Capabilities []string `toml:"capabilities"`
	Modules      []string `toml:"modules"`
}

func (cfg *Config) apply(file configFile) {
	if len(file.Plugins) == 0 {
		return
	}
	cfg.Plugins = make(map[string]PluginConfig, len(file.Plugins))
	for name, plugin := range file.Plugins {
		autoStart := true
		if plugin.AutoStart != nil {
			autoStart = *plugin.AutoStart
		}
		cfg.Plugins[name] = PluginConfig{
			Command:      plugin.Command,
			Args:         plugin.Args,
			Cwd:          plugin.Cwd,
			AutoStart:    autoStart,
			Capabilities: plugin.Capabilities,
			Modules:      plugin.Modules,
		}
	}
}
