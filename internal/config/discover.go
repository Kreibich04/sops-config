package config

import (
	"fmt"
	"io/fs"
	"path/filepath"
)

// FileName is the name every merged config file must have.
const FileName = "sops-config.yaml"

// Discovered pairs a parsed Config with the directory (relative to the
// discovery root, "/"-separated, "." for the root itself) it was found in.
type Discovered struct {
	Config *Config
	Dir    string
	IsRoot bool
}

// Discover walks root looking for every sops-config.yaml file, in stable
// lexical order (root's own config first, then subdirectories in walk
// order). It requires exactly one config at the root directory, since that
// is the baseline the rest of the tree extends.
func Discover(root string) ([]Discovered, error) {
	var found []Discovered

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if d.Name() == ".git" {
				return fs.SkipDir
			}
			return nil
		}
		if d.Name() != FileName {
			return nil
		}

		cfg, loadErr := LoadFile(path)
		if loadErr != nil {
			return loadErr
		}

		relDir, relErr := filepath.Rel(root, filepath.Dir(path))
		if relErr != nil {
			return relErr
		}
		relDir = filepath.ToSlash(relDir)

		found = append(found, Discovered{
			Config: cfg,
			Dir:    relDir,
			IsRoot: relDir == ".",
		})
		return nil
	})
	if err != nil {
		return nil, err
	}

	hasRoot := false
	for _, d := range found {
		if d.IsRoot {
			hasRoot = true
			break
		}
	}
	if !hasRoot {
		return nil, fmt.Errorf("no %s found at root %q: a baseline config is required", FileName, root)
	}

	return found, nil
}
