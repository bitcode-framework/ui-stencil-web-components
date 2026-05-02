package python

import (
	"context"
	"fmt"
	"os/exec"
	goruntime "runtime"
	"strings"

	runtimes "github.com/bitcode-framework/go-json-runtimes"
)

type PythonRuntime struct {
	config runtimes.PythonConfig
}

func New(opts ...runtimes.PythonConfig) *PythonRuntime {
	cfg := runtimes.PythonConfig{
		Enabled:    "auto",
		Command:    "python3",
		MinVersion: "3.10.0",
	}
	if len(opts) > 0 {
		cfg = opts[0]
	}
	return &PythonRuntime{config: cfg}
}

func NewAuto() *PythonRuntime {
	return New()
}

func (p *PythonRuntime) Name() string { return "python" }

func (p *PythonRuntime) Execute(ctx context.Context, script string, function string, params map[string]any, bridge map[string]any) (any, error) {
	if err := p.Validate(); err != nil {
		return nil, err
	}
	return nil, fmt.Errorf("python runtime: direct execution not yet implemented (use BitCode plugin manager for full Python support)")
}

func (p *PythonRuntime) Validate() error {
	if p.config.Enabled == "false" {
		return fmt.Errorf("Python runtime disabled by configuration")
	}

	command := p.config.Command
	if command == "" {
		command = detectPythonCommand()
	}

	_, err := exec.LookPath(command)
	if err != nil {
		if p.config.Enabled == "true" {
			return fmt.Errorf("Python required but not found: %w", err)
		}
		return fmt.Errorf("Python not found in PATH")
	}

	version, err := getPythonVersion(command)
	if err != nil {
		return fmt.Errorf("failed to get Python version: %w", err)
	}

	if !meetsMinVersion(version, p.config.MinVersion) {
		return fmt.Errorf("Python %s required, found %s", p.config.MinVersion, version)
	}

	return nil
}

func (p *PythonRuntime) Close() error {
	return nil
}

func detectPythonCommand() string {
	if goruntime.GOOS == "windows" {
		if _, err := exec.LookPath("python"); err == nil {
			return "python"
		}
	}
	if _, err := exec.LookPath("python3"); err == nil {
		return "python3"
	}
	return "python"
}

func getPythonVersion(command string) (string, error) {
	out, err := exec.Command(command, "--version").Output()
	if err != nil {
		return "", err
	}
	version := strings.TrimSpace(string(out))
	version = strings.TrimPrefix(version, "Python ")
	return version, nil
}

func meetsMinVersion(current, minimum string) bool {
	currentParts := strings.Split(current, ".")
	minimumParts := strings.Split(minimum, ".")

	for i := 0; i < len(minimumParts) && i < len(currentParts); i++ {
		var c, m int
		fmt.Sscanf(currentParts[i], "%d", &c)
		fmt.Sscanf(minimumParts[i], "%d", &m)
		if c > m {
			return true
		}
		if c < m {
			return false
		}
	}
	return true
}
