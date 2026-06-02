package proj

import (
	"bufio"
	"os"
	"strings"
)

// Config holds project configuration from project.toml.
type Config struct {
	Name    string
	Version string
	Entry   string // entry script file, default "main.gs"
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

	var inProject bool
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if line == "[project]" {
			inProject = true
			continue
		}
		if strings.HasPrefix(line, "[") {
			inProject = false
			continue
		}
		if !inProject {
			continue
		}
		k, v, ok := parseKV(line)
		if !ok {
			continue
		}
		switch k {
		case "name":
			cfg.Name = unquote(v)
		case "version":
			cfg.Version = unquote(v)
		case "entry":
			cfg.Entry = unquote(v)
		}
	}
	return cfg
}

func parseKV(line string) (k, v string, ok bool) {
	// strip inline comment
	if idx := strings.IndexByte(line, '#'); idx >= 0 {
		line = strings.TrimSpace(line[:idx])
	}
	idx := strings.IndexByte(line, '=')
	if idx < 0 {
		return "", "", false
	}
	k = strings.TrimSpace(line[:idx])
	v = strings.TrimSpace(line[idx+1:])
	return k, v, true
}

func unquote(s string) string {
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		return s[1 : len(s)-1]
	}
	return s
}
