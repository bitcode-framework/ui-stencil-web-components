package server

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bitcode-framework/go-json/lang"
)

// StaticFileConfig holds resolved static file serving configuration.
type StaticFileConfig struct {
	Dir    string
	Prefix string
}

// ResolveStaticConfig extracts static file config from ServerConfig.
func ResolveStaticConfig(cfg *lang.ServerConfig) *StaticFileConfig {
	if cfg.Static == nil {
		return nil
	}

	switch v := cfg.Static.(type) {
	case string:
		if v == "" {
			return nil
		}
		return &StaticFileConfig{
			Dir:    v,
			Prefix: "/static",
		}
	case lang.StaticConfig:
		if v.Dir == "" {
			return nil
		}
		prefix := v.Prefix
		if prefix == "" {
			prefix = "/static"
		}
		if !strings.HasPrefix(prefix, "/") {
			prefix = "/" + prefix
		}
		return &StaticFileConfig{
			Dir:    v.Dir,
			Prefix: prefix,
		}
	}

	return nil
}

// ValidateStaticDir checks that the static directory exists and is a directory.
func ValidateStaticDir(dir string) error {
	info, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("static directory %q does not exist", dir)
		}
		return fmt.Errorf("static directory %q: %w", dir, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("static path %q is not a directory", dir)
	}
	return nil
}

// IsPathSafe checks that a requested file path doesn't escape the static directory.
func IsPathSafe(basedir, requestedPath string) bool {
	absBase, err := filepath.Abs(basedir)
	if err != nil {
		return false
	}
	absReq, err := filepath.Abs(filepath.Join(basedir, requestedPath))
	if err != nil {
		return false
	}

	if !strings.HasPrefix(absReq, absBase) {
		return false
	}

	base := filepath.Base(requestedPath)
	if strings.HasPrefix(base, ".") {
		return false
	}

	return true
}
