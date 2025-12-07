// SPDX-FileCopyrightText: 2024-2025 Rafael V. Volkmer <rafael.v.volkmer@gmail.com>
// SPDX-License-Identifier: MIT

package usecase

import (
	"context"

	"github.com/rafaelvolkmer/codeaudit/internal/domain/model"
)

type ListMetricsUseCase struct{}

func NewListMetricsUseCase() *ListMetricsUseCase {
	return &ListMetricsUseCase{}
}

func (uc *ListMetricsUseCase) Execute(ctx context.Context) []model.MetricSummary {
	_ = ctx
	return model.AllMetricSummaries()
}
