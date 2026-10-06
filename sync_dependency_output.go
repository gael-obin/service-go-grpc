package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
)

// Dependency directories disappear from the generation loop when a dependency
// is removed. Reclaim their marked Go output in the same transaction, preserving
// handwritten files and never following symlinks into another tree.
func trackRetiredDependencyBindings(transaction *syncTransaction, moduleRoot string) error {
	root := filepath.Join(transaction.actualRoot, moduleRoot, "external")
	return filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if os.IsNotExist(walkErr) && path == root {
			return nil
		}
		if walkErr != nil {
			return walkErr
		}
		relative, err := filepath.Rel(transaction.actualRoot, path)
		if err != nil {
			return err
		}
		if target, ok := transaction.targets[relative]; ok && target.directory {
			return filepath.SkipDir // a current dependency is regenerated as a whole
		}
		if !entry.Type().IsRegular() || filepath.Ext(path) != ".go" {
			return nil
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.PackageClauseOnly|parser.ParseComments)
		if err != nil {
			return err
		}
		if ast.IsGenerated(file) {
			return transaction.TrackFile(relative)
		}
		return nil
	})
}
