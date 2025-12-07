// SPDX-FileCopyrightText: 2024-2025 Rafael V. Volkmer <rafael.v.volkmer@gmail.com>
// SPDX-License-Identifier: MIT

package integration

import (
	"context"
	"path/filepath"
	"testing"

	gitadapter "github.com/rafaelvolkmer/codeaudit/internal/adapter/git"
	parser "github.com/rafaelvolkmer/codeaudit/internal/adapter/parser"
	"github.com/rafaelvolkmer/codeaudit/internal/domain/ports"
	"github.com/rafaelvolkmer/codeaudit/internal/infrastructure"
	"github.com/rafaelvolkmer/codeaudit/internal/usecase"
)

func TestAnalyzeSampleProject(t *testing.T) {
	root := filepath.Join("..", "data")
	ctx := context.Background()

	scanner := infrastructure.NewFSScanner()
	storage := infrastructure.NewFileStorage()
	gitClient := gitadapter.NewGitCLI()

	parsers := []ports.CodeParser{
		parser.NewGoParser(),
		parser.NewCParser(),
	}

	uc := usecase.NewAnalyzeProjectUseCase(
		scanner,
		scanner,
		parsers,
		gitClient,
		storage,
		2,
	)

	report, err := uc.Execute(ctx, usecase.AnalyzeProjectRequest{
		RootPath:   root,
		IncludeExt: []string{".go", ".c"},
	})
	if err != nil {
		t.Fatalf("AnalyzeProject failed: %v", err)
	}

	if len(report.Files) == 0 {
		t.Fatalf("expected at least one file in report")
	}
	if report.Project.TotalFunctions == 0 {
		t.Fatalf("expected at least one function in project metrics")
	}
}
