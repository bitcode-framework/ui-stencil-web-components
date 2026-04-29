package io

import (
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

// FSModule provides file system functions for go-json programs.
type FSModule struct {
	security *SecurityConfig
	config   map[string]any
}

// NewFSModule creates a new file system I/O module.
func NewFSModule(security *SecurityConfig) *FSModule {
	if security == nil {
		security = DefaultSecurityConfig()
	}
	return &FSModule{security: security}
}

func (m *FSModule) Name() string { return "fs" }

func (m *FSModule) SetConfig(cfg map[string]any) { m.config = cfg }

func (m *FSModule) Functions() map[string]any {
	return map[string]any{
		"read":   m.fsRead,
		"write":  m.fsWrite,
		"append": m.fsAppend,
		"exists": m.fsExists,
		"list":   m.fsList,
		"mkdir":  m.fsMkdir,
		"remove": m.fsRemove,
		"stat":   m.fsStat,
		"copy":   m.fsCopy,
		"move":   m.fsMove,
		"glob":   m.fsGlob,
	}
}

func (m *FSModule) fsRead(params ...any) (any, error) {
	if len(params) < 1 {
		return nil, fmt.Errorf("fs.read: path is required")
	}
	path, ok := params[0].(string)
	if !ok {
		return nil, fmt.Errorf("fs.read: path must be a string")
	}

	if err := m.security.ValidateFilePath(path, false); err != nil {
		return nil, err
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("fs.read: %s", err.Error())
	}

	// Resolve symlinks and re-validate.
	resolved, err := filepath.EvalSymlinks(absPath)
	if err == nil && resolved != absPath {
		if err := m.security.ValidateFilePath(resolved, false); err != nil {
			return nil, err
		}
		absPath = resolved
	}

	info, err := os.Stat(absPath)
	if err != nil {
		return nil, fmt.Errorf("fs.read: %s", err.Error())
	}
	if info.IsDir() {
		return nil, fmt.Errorf("fs.read: cannot read directory, use fs.list")
	}

	maxSize := m.security.FS.MaxFileSize
	if maxSize > 0 && info.Size() > maxSize {
		return nil, fmt.Errorf("fs.read: file exceeds max size (%d bytes, max %d)", info.Size(), maxSize)
	}

	data, err := os.ReadFile(absPath)
	if err != nil {
		return nil, fmt.Errorf("fs.read: %s", err.Error())
	}

	encoding := "utf-8"
	if len(params) > 1 {
		if enc, ok := params[1].(string); ok {
			encoding = enc
		}
	}

	if encoding == "base64" {
		return base64.StdEncoding.EncodeToString(data), nil
	}

	return string(data), nil
}

func (m *FSModule) fsWrite(params ...any) (any, error) {
	if len(params) < 2 {
		return nil, fmt.Errorf("fs.write: path and content are required")
	}
	path, ok := params[0].(string)
	if !ok {
		return nil, fmt.Errorf("fs.write: path must be a string")
	}
	content, ok := params[1].(string)
	if !ok {
		return nil, fmt.Errorf("fs.write: content must be a string")
	}

	if err := m.security.ValidateFilePath(path, true); err != nil {
		return nil, err
	}

	var data []byte
	encoding := "utf-8"
	if len(params) > 2 {
		if enc, ok := params[2].(string); ok {
			encoding = enc
		}
	}

	if encoding == "base64" {
		var err error
		data, err = base64.StdEncoding.DecodeString(content)
		if err != nil {
			return nil, fmt.Errorf("fs.write: invalid base64 content: %s", err.Error())
		}
	} else {
		data = []byte(content)
	}

	maxSize := m.security.FS.MaxFileSize
	if maxSize > 0 && int64(len(data)) > maxSize {
		return nil, fmt.Errorf("fs.write: content exceeds max size (%d bytes, max %d)", len(data), maxSize)
	}

	absPath, _ := filepath.Abs(path)
	if err := os.WriteFile(absPath, data, 0644); err != nil {
		return nil, fmt.Errorf("fs.write: %s", err.Error())
	}

	return nil, nil
}

func (m *FSModule) fsAppend(params ...any) (any, error) {
	if len(params) < 2 {
		return nil, fmt.Errorf("fs.append: path and content are required")
	}
	path, ok := params[0].(string)
	if !ok {
		return nil, fmt.Errorf("fs.append: path must be a string")
	}
	content, ok := params[1].(string)
	if !ok {
		return nil, fmt.Errorf("fs.append: content must be a string")
	}

	if err := m.security.ValidateFilePath(path, true); err != nil {
		return nil, err
	}

	absPath, _ := filepath.Abs(path)
	f, err := os.OpenFile(absPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("fs.append: %s", err.Error())
	}
	defer f.Close()

	if _, err := f.WriteString(content); err != nil {
		return nil, fmt.Errorf("fs.append: %s", err.Error())
	}

	return nil, nil
}

func (m *FSModule) fsExists(params ...any) (any, error) {
	if len(params) < 1 {
		return nil, fmt.Errorf("fs.exists: path is required")
	}
	path, ok := params[0].(string)
	if !ok {
		return nil, fmt.Errorf("fs.exists: path must be a string")
	}

	if err := m.security.ValidateFilePath(path, false); err != nil {
		return nil, err
	}

	absPath, _ := filepath.Abs(path)
	_, err := os.Stat(absPath)
	return err == nil, nil
}

func (m *FSModule) fsList(params ...any) (any, error) {
	if len(params) < 1 {
		return nil, fmt.Errorf("fs.list: path is required")
	}
	path, ok := params[0].(string)
	if !ok {
		return nil, fmt.Errorf("fs.list: path must be a string")
	}

	if err := m.security.ValidateFilePath(path, false); err != nil {
		return nil, err
	}

	absPath, _ := filepath.Abs(path)
	entries, err := os.ReadDir(absPath)
	if err != nil {
		return nil, fmt.Errorf("fs.list: %s", err.Error())
	}

	names := make([]any, len(entries))
	for i, entry := range entries {
		names[i] = entry.Name()
	}
	return names, nil
}

func (m *FSModule) fsMkdir(params ...any) (any, error) {
	if len(params) < 1 {
		return nil, fmt.Errorf("fs.mkdir: path is required")
	}
	path, ok := params[0].(string)
	if !ok {
		return nil, fmt.Errorf("fs.mkdir: path must be a string")
	}

	if err := m.security.ValidateFilePath(path, true); err != nil {
		return nil, err
	}

	absPath, _ := filepath.Abs(path)
	if err := os.MkdirAll(absPath, 0755); err != nil {
		return nil, fmt.Errorf("fs.mkdir: %s", err.Error())
	}

	return nil, nil
}

func (m *FSModule) fsRemove(params ...any) (any, error) {
	if len(params) < 1 {
		return nil, fmt.Errorf("fs.remove: path is required")
	}
	path, ok := params[0].(string)
	if !ok {
		return nil, fmt.Errorf("fs.remove: path must be a string")
	}

	if err := m.security.ValidateFilePath(path, true); err != nil {
		return nil, err
	}

	absPath, _ := filepath.Abs(path)

	info, err := os.Stat(absPath)
	if err != nil {
		return nil, fmt.Errorf("fs.remove: %s", err.Error())
	}

	recursive := false
	if len(params) > 1 {
		if r, ok := params[1].(bool); ok {
			recursive = r
		}
	}

	if info.IsDir() && !recursive {
		entries, _ := os.ReadDir(absPath)
		if len(entries) > 0 {
			return nil, fmt.Errorf("fs.remove: directory not empty, use recursive: true")
		}
	}

	if recursive {
		err = os.RemoveAll(absPath)
	} else {
		err = os.Remove(absPath)
	}
	if err != nil {
		return nil, fmt.Errorf("fs.remove: %s", err.Error())
	}

	return nil, nil
}

func (m *FSModule) fsStat(params ...any) (any, error) {
	if len(params) < 1 {
		return nil, fmt.Errorf("fs.stat: path is required")
	}
	path, ok := params[0].(string)
	if !ok {
		return nil, fmt.Errorf("fs.stat: path must be a string")
	}

	if err := m.security.ValidateFilePath(path, false); err != nil {
		return nil, err
	}

	absPath, _ := filepath.Abs(path)
	info, err := os.Stat(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("fs.stat: file not found: %s", path)
		}
		return nil, fmt.Errorf("fs.stat: %s", err.Error())
	}

	return map[string]any{
		"name":        info.Name(),
		"size":        info.Size(),
		"is_dir":      info.IsDir(),
		"is_file":     !info.IsDir(),
		"ext":         filepath.Ext(info.Name()),
		"modified":    info.ModTime().Format(time.RFC3339),
		"permissions": info.Mode().String(),
	}, nil
}

func (m *FSModule) fsCopy(params ...any) (any, error) {
	if len(params) < 2 {
		return nil, fmt.Errorf("fs.copy: src and dst are required")
	}
	src, ok := params[0].(string)
	if !ok {
		return nil, fmt.Errorf("fs.copy: src must be a string")
	}
	dst, ok := params[1].(string)
	if !ok {
		return nil, fmt.Errorf("fs.copy: dst must be a string")
	}

	if err := m.security.ValidateFilePath(src, false); err != nil {
		return nil, err
	}
	if err := m.security.ValidateFilePath(dst, true); err != nil {
		return nil, err
	}

	absSrc, _ := filepath.Abs(src)
	absDst, _ := filepath.Abs(dst)

	srcFile, err := os.Open(absSrc)
	if err != nil {
		return nil, fmt.Errorf("fs.copy: %s", err.Error())
	}
	defer srcFile.Close()

	dstFile, err := os.Create(absDst)
	if err != nil {
		return nil, fmt.Errorf("fs.copy: %s", err.Error())
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		return nil, fmt.Errorf("fs.copy: %s", err.Error())
	}

	return nil, nil
}

func (m *FSModule) fsMove(params ...any) (any, error) {
	if len(params) < 2 {
		return nil, fmt.Errorf("fs.move: src and dst are required")
	}
	src, ok := params[0].(string)
	if !ok {
		return nil, fmt.Errorf("fs.move: src must be a string")
	}
	dst, ok := params[1].(string)
	if !ok {
		return nil, fmt.Errorf("fs.move: dst must be a string")
	}

	if err := m.security.ValidateFilePath(src, true); err != nil {
		return nil, err
	}
	if err := m.security.ValidateFilePath(dst, true); err != nil {
		return nil, err
	}

	absSrc, _ := filepath.Abs(src)
	absDst, _ := filepath.Abs(dst)

	if err := os.Rename(absSrc, absDst); err != nil {
		return nil, fmt.Errorf("fs.move: %s", err.Error())
	}

	return nil, nil
}

func (m *FSModule) fsGlob(params ...any) (any, error) {
	if len(params) < 1 {
		return nil, fmt.Errorf("fs.glob: pattern is required")
	}
	pattern, ok := params[0].(string)
	if !ok {
		return nil, fmt.Errorf("fs.glob: pattern must be a string")
	}

	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("fs.glob: %s", err.Error())
	}

	result := make([]any, len(matches))
	for i, m := range matches {
		result[i] = filepath.ToSlash(m)
	}

	return result, nil
}
