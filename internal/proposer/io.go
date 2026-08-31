package proposer

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

func ReadJSON(path string, target any) ([]byte, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return nil, fmt.Errorf("decode %s: %w", path, err)
	}
	return raw, nil
}

func WriteJSON(path string, value any) ([]byte, error) {
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return nil, err
	}
	raw = append(raw, '\n')
	if err := WriteBytes(path, raw); err != nil {
		return nil, err
	}
	return raw, nil
}

func WriteBytes(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".gooo-proposer-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if err := tmp.Chmod(0o644); err != nil {
		_ = tmp.Close()
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}

func EnsureCallerOwnedOutput(path string) error {
	if path == "" || !filepath.IsAbs(path) {
		return fmt.Errorf("output must be an existing absolute temporary directory")
	}
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("output directory: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("output is not a directory")
	}
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		return err
	}
	working, err := os.Getwd()
	if err != nil {
		return err
	}
	working, err = filepath.EvalSymlinks(working)
	if err != nil {
		return err
	}
	if root := findGitRoot(working); root != "" && pathWithin(root, resolved) {
		return fmt.Errorf("output may not be inside the input repository")
	}
	return nil
}

func findGitRoot(start string) string {
	current := start
	for {
		if info, err := os.Stat(filepath.Join(current, ".git")); err == nil && (info.IsDir() || info.Mode().IsRegular()) {
			return current
		}
		parent := filepath.Dir(current)
		if parent == current {
			return ""
		}
		current = parent
	}
}

func pathWithin(root, path string) bool {
	relative, err := filepath.Rel(root, path)
	return err == nil && relative != ".." && len(relative) > 3 && relative[:3] != ".."+string(filepath.Separator)
}
