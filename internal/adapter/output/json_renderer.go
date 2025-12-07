// SPDX-FileCopyrightText: 2024-2025 Rafael V. Volkmer <rafael.v.volkmer@gmail.com>
// SPDX-License-Identifier: MIT

package output

import (
	"encoding/json"

	"github.com/rafaelvolkmer/codeaudit/internal/domain/model"
	"github.com/rafaelvolkmer/codeaudit/internal/domain/ports"
)

type JSONRenderer struct{}

func NewJSONRenderer() *JSONRenderer {
	return &JSONRenderer{}
}

var _ ports.OutputRenderer = (*JSONRenderer)(nil)

func (r *JSONRenderer) Format() string {
	return "json"
}

func (r *JSONRenderer) Render(report *model.ProjectReport) (string, error) {
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}
