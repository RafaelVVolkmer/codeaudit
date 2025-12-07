// SPDX-FileCopyrightText: 2024-2025 Rafael V. Volkmer <rafael.v.volkmer@gmail.com>
// SPDX-License-Identifier: MIT

package parser

import (
	"regexp"
	"strings"
)

var decisionKeywords = regexp.MustCompile(`\b(if|for|while|case|switch)\b`)

var boolOps = regexp.MustCompile(`&&|\|\||\?`)

func estimateCommentLines(lines []string) int {
	inBlock := false
	count := 0

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		if inBlock {
			count++
			if strings.Contains(trimmed, "*/") {
				inBlock = false
			}
			continue
		}

		if strings.HasPrefix(trimmed, "//") {
			count++
			continue
		}

		if idx := strings.Index(trimmed, "/*"); idx >= 0 {
			count++
			if !strings.Contains(trimmed[idx+2:], "*/") {
				inBlock = true
			}
		}
	}

	return count
}

func computeTextMetricsForRange(lines []string, startLine, endLine int) (
	nloc int,
	ccn int,
	cognitive int,
	maxNesting int,
	locals int,
	commentLines int,
) {
	if startLine < 1 {
		startLine = 1
	}
	if endLine > len(lines) {
		endLine = len(lines)
	}

	ccn = 1
	blockDepth := 0
	inBlockComment := false

	for i := startLine - 1; i < endLine; i++ {
		line := lines[i]
		trimmed := strings.TrimSpace(line)

		if trimmed == "" {
			continue
		}

		if inBlockComment {
			commentLines++
			if strings.Contains(trimmed, "*/") {
				inBlockComment = false
			}
			continue
		}

		if idx := strings.Index(trimmed, "/*"); idx >= 0 {
			commentLines++
			if !strings.Contains(trimmed[idx+2:], "*/") {
				inBlockComment = true
				continue
			}

			trimmed = strings.TrimSpace(trimmed[:idx])
			if trimmed == "" {
				continue
			}
		}

		if idx := strings.Index(trimmed, "//"); idx >= 0 {
			if strings.TrimSpace(trimmed[:idx]) == "" {
				commentLines++
				continue
			}

			commentLines++
			trimmed = strings.TrimSpace(trimmed[:idx])
			if trimmed == "" {
				continue
			}
		}

		if strings.HasPrefix(trimmed, "#") {
			continue
		}

		nloc++

		code := stripStringLiterals(trimmed)

		decisions := len(decisionKeywords.FindAllStringSubmatch(code, -1))
		bools := len(boolOps.FindAllStringSubmatch(code, -1))

		ccn += decisions + bools

		opens := strings.Count(code, "{")
		closes := strings.Count(code, "}")
		blockDepth += opens - closes
		if blockDepth < 0 {
			blockDepth = 0
		}
		if blockDepth > maxNesting {
			maxNesting = blockDepth
		}

		cognitive += decisions + blockDepth
	}

	return
}

func stripStringLiterals(s string) string {
	var b strings.Builder
	inSingle := false
	inDouble := false
	escape := false

	for _, r := range s {
		if escape {
			escape = false
			continue
		}

		switch r {
		case '\\':
			if inSingle || inDouble {
				escape = true
			} else {
				b.WriteRune(r)
			}
		case '\'':
			if !inDouble {
				inSingle = !inSingle
			} else {
				b.WriteRune(r)
			}
		case '"':
			if !inSingle {
				inDouble = !inDouble
			} else {
				b.WriteRune(r)
			}
		default:
			if !inSingle && !inDouble {
				b.WriteRune(r)
			}
		}
	}

	return b.String()
}
