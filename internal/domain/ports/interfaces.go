// SPDX-FileCopyrightText: 2024-2025 Rafael V. Volkmer <rafael.v.volkmer@gmail.com>
// SPDX-License-Identifier: MIT

package ports

import (
	"context"

	"github.com/rafaelvolkmer/codeaudit/internal/domain/model"
)

type SourceFileScanner interface {
	Scan(ctx context.Context, root string, includeExt []string) ([]string, error)
}

type FileReader interface {
	ReadFile(path string) ([]byte, error)
}

type CodeParser interface {
	Name() string
	SupportsFile(path string) bool
	ParseFile(path string, src []byte) (*model.FileMetrics, error)
}

type GitClient interface {
	CollectFileMetrics(ctx context.Context, root string) (map[string]*model.GitFileMetrics, error)
}

type ReportStorage interface {
	Save(ctx context.Context, root string, report *model.ProjectReport) error
	Load(ctx context.Context, root string) (*model.ProjectReport, error)
}

type OutputRenderer interface {
	Format() string
	Render(report *model.ProjectReport) (string, error)
}

type RendererRegistry interface {
	Get(format string) (OutputRenderer, bool)
	List() []OutputRenderer
}
