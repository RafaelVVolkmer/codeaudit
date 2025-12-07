// SPDX-FileCopyrightText: 2024-2025 Rafael V. Volkmer <rafael.v.volkmer@gmail.com>
// SPDX-License-Identifier: MIT

package infrastructure

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/rafaelvolkmer/codeaudit/internal/domain/ports"
)

type FSScanner struct{}

func NewFSScanner() *FSScanner {
	return &FSScanner{}
}

var _ ports.SourceFileScanner = (*FSScanner)(nil)
var _ ports.FileReader = (*FSScanner)(nil)

func (s *FSScanner) Scan(ctx context.Context, root string, includeExt []string) ([]string, error) {
	var files []string

	allowed := make(map[string]struct{}, len(includeExt))
	for _, e := range includeExt {
		allowed[strings.ToLower(e)] = struct{}{}
	}

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			name := d.Name()
			switch name {
			case ".git", "vendor", "node_modules", ".codeaudit":
				return filepath.SkipDir
			default:
				return nil
			}
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if !d.Type().IsRegular() {
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		if len(allowed) > 0 {
			if _, ok := allowed[ext]; !ok {
				return nil
			}
		}

		files = append(files, path)
		return nil
	})

	return files, err
}

func (s *FSScanner) ReadFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}
