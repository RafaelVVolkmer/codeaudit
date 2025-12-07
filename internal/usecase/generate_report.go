// SPDX-FileCopyrightText: 2024-2025 Rafael V. Volkmer <rafael.v.volkmer@gmail.com>
// SPDX-License-Identifier: MIT

package usecase

import (
	"context"
	"fmt"
	"strings"

	"github.com/rafaelvolkmer/codeaudit/internal/domain/ports"
)

type GenerateReportRequest struct {
	RootPath string
	Format   string
}

type GenerateReportUseCase struct {
	storage  ports.ReportStorage
	registry ports.RendererRegistry
}

func NewGenerateReportUseCase(storage ports.ReportStorage, registry ports.RendererRegistry) *GenerateReportUseCase {
	return &GenerateReportUseCase{
		storage:  storage,
		registry: registry,
	}
}

func (uc *GenerateReportUseCase) Execute(ctx context.Context, req GenerateReportRequest) (string, error) {
	report, err := uc.storage.Load(ctx, req.RootPath)
	if err != nil {
		return "", err
	}

	format := strings.ToLower(req.Format)
	if format == "" {
		format = "text"
	}

	renderer, ok := uc.registry.Get(format)
	if !ok {
		return "", fmt.Errorf("unknown format %q", format)
	}

	return renderer.Render(report)
}
