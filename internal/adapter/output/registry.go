// SPDX-FileCopyrightText: 2024-2025 Rafael V. Volkmer <rafael.v.volkmer@gmail.com>
// SPDX-License-Identifier: MIT

package output

import (
	"strings"

	"github.com/rafaelvolkmer/codeaudit/internal/domain/ports"
)

type RendererRegistry struct {
	byFormat map[string]ports.OutputRenderer
}

func NewRendererRegistry(renderers ...ports.OutputRenderer) *RendererRegistry {
	m := make(map[string]ports.OutputRenderer, len(renderers))
	for _, r := range renderers {
		if r == nil {
			continue
		}
		m[strings.ToLower(r.Format())] = r
	}
	return &RendererRegistry{byFormat: m}
}

var _ ports.RendererRegistry = (*RendererRegistry)(nil)

func (r *RendererRegistry) Get(format string) (ports.OutputRenderer, bool) {
	if r == nil {
		return nil, false
	}
	f := strings.ToLower(format)
	out, ok := r.byFormat[f]
	return out, ok
}

func (r *RendererRegistry) List() []ports.OutputRenderer {
	out := make([]ports.OutputRenderer, 0, len(r.byFormat))
	for _, v := range r.byFormat {
		out = append(out, v)
	}
	return out
}
