package node

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	runtimes "github.com/bitcode-framework/go-json-runtimes"
)

type NodeRuntime struct {
	config runtimes.NodeConfig
}

func New(opts ...runtimes.NodeConfig) *NodeRuntime {
	cfg := runtimes.NodeConfig{
		Enabled:    "auto",
		Command:    "node",
		MinVersion: "20.0",
	}
	if len(opts) > 0 {
		cfg = opts[0]
	}
	return &NodeRuntime{config: cfg}
}

func NewAuto() *NodeRuntime {
	return New()
}

func (n *NodeRuntime) Name() string { return "node" }

func (n *NodeRuntime) Execute(ctx context.Context, script string, function string, params map[string]any, bridge map[string]any) (any, error) {
	if err := n.Validate(); err != nil {
		return nil, err
	}
	return nil, fmt.Errorf("node runtime: direct execution not yet implemented (use BitCode plugin manager for full Node.js support)")
}

func (n *NodeRuntime) Validate() error {
	if n.config.Enabled == "false" {
		return fmt.Errorf("Node.js runtime disabled by configuration")
	}

	command := n.config.Command
	if command == "" {
		command = "node"
	}

	_, err := exec.LookPath(command)
	if err != nil {
		if n.config.Enabled == "true" {
			return fmt.Errorf("Node.js required but not found: %w", err)
		}
		return fmt.Errorf("Node.js not found in PATH")
	}

	version, err := getNodeVersion(command)
	if err != nil {
		return fmt.Errorf("failed to get Node.js version: %w", err)
	}

	if !meetsMinVersion(version, n.config.MinVersion) {
		return fmt.Errorf("Node.js %s required, found %s", n.config.MinVersion, version)
	}

	return nil
}

func (n *NodeRuntime) Close() error {
	return nil
}

func getNodeVersion(command string) (string, error) {
	out, err := exec.Command(command, "--version").Output()
	if err != nil {
		return "", err
	}
	version := strings.TrimSpace(string(out))
	version = strings.TrimPrefix(version, "v")
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
