package proj

import (
	"bufio"
	"os"
	"strconv"
	"strings"
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

// Load reads a project.toml file and returns parsed config.
// Returns defaults if the file doesn't exist.
func Load(path string) *Config {
	cfg := &Config{
		Name:  "project",
		Entry: "main.gs",
	}

	f, err := os.Open(path)
	if err != nil {
		return cfg
	}
	defer f.Close()

	var section string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if line == "[project]" {
			section = "project"
			continue
		}
		if strings.HasPrefix(line, "[") {
			section = strings.TrimSpace(strings.Trim(line, "[]"))
			continue
		}
		k, v, ok := parseKV(line)
		if !ok {
			continue
		}
		switch section {
		case "project":
			switch k {
			case "name":
				cfg.Name = parseString(v)
			case "version":
				cfg.Version = parseString(v)
			case "entry":
				cfg.Entry = parseString(v)
			}
		case "package":
			switch k {
			case "name":
				cfg.Package.Name = parseString(v)
			case "version":
				cfg.Package.Version = parseString(v)
			case "type":
				cfg.Package.Type = parseString(v)
			case "main":
				cfg.Package.Main = parseString(v)
			}
		case "exports":
			if cfg.Exports == nil {
				cfg.Exports = make(map[string]string)
			}
			cfg.Exports[parseString(k)] = parseString(v)
		case "imports":
			if cfg.Imports == nil {
				cfg.Imports = make(map[string]string)
			}
			cfg.Imports[parseString(k)] = parseString(v)
		case "dependencies":
			if cfg.Dependencies == nil {
				cfg.Dependencies = make(map[string]string)
			}
			cfg.Dependencies[parseString(k)] = parseString(v)
		case "bundle":
			switch k {
			case "target":
				cfg.Bundle.Target = parseString(v)
			case "format":
				cfg.Bundle.Format = parseString(v)
			case "includeStd":
				cfg.Bundle.IncludeStd = parseBool(v)
			case "external":
				cfg.Bundle.External = parseStringArray(v)
			}
		}
	}
	return cfg
}

func parseKV(line string) (k, v string, ok bool) {
	line = stripInlineComment(line)
	idx := strings.IndexByte(line, '=')
	if idx < 0 {
		return "", "", false
	}
	k = strings.TrimSpace(line[:idx])
	v = strings.TrimSpace(line[idx+1:])
	return k, v, true
}

func stripInlineComment(line string) string {
	inString := false
	for i := 0; i < len(line); i++ {
		switch line[i] {
		case '"':
			if i == 0 || line[i-1] != '\\' {
				inString = !inString
			}
		case '#':
			if !inString {
				return strings.TrimSpace(line[:i])
			}
		}
	}
	return line
}

func parseString(s string) string {
	return unquote(s)
}

func parseBool(s string) bool {
	v, err := strconv.ParseBool(strings.ToLower(strings.TrimSpace(s)))
	return err == nil && v
}

func parseStringArray(s string) []string {
	s = strings.TrimSpace(s)
	if len(s) < 2 || s[0] != '[' || s[len(s)-1] != ']' {
		return nil
	}
	s = strings.TrimSpace(s[1 : len(s)-1])
	if s == "" {
		return []string{}
	}
	parts := strings.Split(s, ",")
	values := make([]string, 0, len(parts))
	for _, part := range parts {
		value := parseString(strings.TrimSpace(part))
		if value != "" {
			values = append(values, value)
		}
	}
	return values
}

func unquote(s string) string {
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		return s[1 : len(s)-1]
	}
	return s
}
