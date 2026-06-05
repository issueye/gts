package proj

import (
	"fmt"
	"os"

	"github.com/pelletier/go-toml/v2"
)

// Config holds project configuration from project.toml.
type Config struct {
	Name         string
	Version      string
	Entry        string // entry script file, default "main.gs"
	Package      PackageConfig
	Exports      map[string]string
	Imports      map[string]string
	Dependencies map[string]string
	Bundle       BundleConfig
	Plugins      map[string]PluginConfig
}

// PackageConfig holds package metadata from the [package] section.
type PackageConfig struct {
	Name    string
	Version string
	Type    string
	Main    string
}

// BundleConfig holds bundling options from the [bundle] section.
type BundleConfig struct {
	Target     string
	Format     string
	IncludeStd bool
	External   []string
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

// Load reads a project.toml file and returns parsed config.
// Returns defaults if the file doesn't exist.
func Load(path string) *Config {
	cfg, _ := LoadStrict(path)
	return cfg
}

// LoadStrict reads a project.toml file and reports invalid TOML. Missing files
// still return the default config so single-file projects can run without a
// manifest.
func LoadStrict(path string) (*Config, error) {
	cfg := &Config{
		Name:  "project",
		Entry: "main.gs",
	}

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
	cfg := &Config{
		Name:  "project",
		Entry: "main.gs",
	}
	var manifest manifestFile
	if err := toml.Unmarshal([]byte(src), &manifest); err != nil {
		return cfg, fmt.Errorf("invalid project manifest %q: %w", name, err)
	}
	cfg.applyManifest(manifest)
	return cfg, nil
}

type manifestFile struct {
	Project      projectSection           `toml:"project"`
	Package      packageSection           `toml:"package"`
	Exports      map[string]string        `toml:"exports"`
	Imports      map[string]string        `toml:"imports"`
	Dependencies map[string]string        `toml:"dependencies"`
	Bundle       bundleSection            `toml:"bundle"`
	Plugins      map[string]pluginSection `toml:"plugins"`
}

type projectSection struct {
	Name    string `toml:"name"`
	Version string `toml:"version"`
	Entry   string `toml:"entry"`
}

type packageSection struct {
	Name    string `toml:"name"`
	Version string `toml:"version"`
	Type    string `toml:"type"`
	Main    string `toml:"main"`
}

type bundleSection struct {
	Target     string   `toml:"target"`
	Format     string   `toml:"format"`
	IncludeStd bool     `toml:"includeStd"`
	External   []string `toml:"external"`
}

type pluginSection struct {
	Command      string   `toml:"command"`
	Args         []string `toml:"args"`
	Cwd          string   `toml:"cwd"`
	AutoStart    *bool    `toml:"autoStart"`
	Capabilities []string `toml:"capabilities"`
	Modules      []string `toml:"modules"`
}

func (cfg *Config) applyManifest(m manifestFile) {
	if m.Project.Name != "" {
		cfg.Name = m.Project.Name
	}
	if m.Project.Version != "" {
		cfg.Version = m.Project.Version
	}
	if m.Project.Entry != "" {
		cfg.Entry = m.Project.Entry
	}
	cfg.Package = PackageConfig{
		Name:    m.Package.Name,
		Version: m.Package.Version,
		Type:    m.Package.Type,
		Main:    m.Package.Main,
	}
	cfg.Exports = m.Exports
	cfg.Imports = m.Imports
	cfg.Dependencies = m.Dependencies
	cfg.Bundle = BundleConfig{
		Target:     m.Bundle.Target,
		Format:     m.Bundle.Format,
		IncludeStd: m.Bundle.IncludeStd,
		External:   m.Bundle.External,
	}
	if len(m.Plugins) > 0 {
		cfg.Plugins = make(map[string]PluginConfig, len(m.Plugins))
		for name, plugin := range m.Plugins {
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
}
