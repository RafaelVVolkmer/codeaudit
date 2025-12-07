// SPDX-FileCopyrightText: 2024-2025 Rafael V. Volkmer <rafael.v.volkmer@gmail.com>
// SPDX-License-Identifier: MIT

package infrastructure

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/rafaelvolkmer/codeaudit/internal/domain/model"
	"github.com/rafaelvolkmer/codeaudit/internal/domain/ports"
)

type FileStorage struct{}

func NewFileStorage() *FileStorage {
	return &FileStorage{}
}

var _ ports.ReportStorage = (*FileStorage)(nil)

func (s *FileStorage) Save(ctx context.Context, root string, report *model.ProjectReport) error {
	_ = ctx

	dir := filepath.Join(root, ".codeaudit")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create report dir: %w", err)
	}
	path := filepath.Join(dir, "report.json")

	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create report file: %w", err)
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	if err := enc.Encode(report); err != nil {
		return fmt.Errorf("encode report: %w", err)
	}
	return nil
}

func (s *FileStorage) Load(ctx context.Context, root string) (*model.ProjectReport, error) {
	_ = ctx

	path := filepath.Join(root, ".codeaudit", "report.json")
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open report: %w", err)
	}
	defer f.Close()

	var report model.ProjectReport
	dec := json.NewDecoder(f)
	if err := dec.Decode(&report); err != nil {
		return nil, fmt.Errorf("decode report: %w", err)
	}
	return &report, nil
}
