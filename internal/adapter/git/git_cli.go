// SPDX-FileCopyrightText: 2024-2025 Rafael V. Volkmer <rafael.v.volkmer@gmail.com>
// SPDX-License-Identifier: MIT

package gitadapter

import (
	"bufio"
	"bytes"
	"context"
	"os/exec"
	"strconv"
	"strings"

	"github.com/rafaelvolkmer/codeaudit/internal/domain/model"
	"github.com/rafaelvolkmer/codeaudit/internal/domain/ports"
)

type GitCLI struct{}

func NewGitCLI() *GitCLI {
	return &GitCLI{}
}

var _ ports.GitClient = (*GitCLI)(nil)

func (g *GitCLI) CollectFileMetrics(ctx context.Context, root string) (map[string]*model.GitFileMetrics, error) {
	cmd := exec.CommandContext(ctx, "git", "-C", root, "log", "--numstat", "--format=commit:%H:%an:%s")
	out, err := cmd.Output()
	if err != nil {
		return map[string]*model.GitFileMetrics{}, nil
	}

	type agg struct {
		added, deleted, commits, bugfixCommits int
		authors                                map[string]struct{}
	}

	aggs := make(map[string]*agg)
	var currentAuthor string
	var currentSubject string
	var isBugfix bool

	scanner := bufio.NewScanner(bytes.NewReader(out))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "commit:") {
			parts := strings.SplitN(line, ":", 4)
			if len(parts) >= 4 {
				currentAuthor = parts[2]
				currentSubject = parts[3]
				lower := strings.ToLower(currentSubject)
				isBugfix = strings.Contains(lower, "fix") ||
					strings.Contains(lower, "bug") ||
					strings.Contains(lower, "issue")
			}
			continue
		}

		fields := strings.Fields(line)
		if len(fields) != 3 {
			continue
		}
		addStr, delStr, path := fields[0], fields[1], fields[2]
		if addStr == "-" || delStr == "-" {
			continue
		}
		added, err1 := strconv.Atoi(addStr)
		deleted, err2 := strconv.Atoi(delStr)
		if err1 != nil || err2 != nil {
			continue
		}

		a := aggs[path]
		if a == nil {
			a = &agg{authors: make(map[string]struct{})}
			aggs[path] = a
		}
		a.added += added
		a.deleted += deleted
		a.commits++
		if currentAuthor != "" {
			a.authors[currentAuthor] = struct{}{}
		}
		if isBugfix {
			a.bugfixCommits++
		}
	}

	result := make(map[string]*model.GitFileMetrics, len(aggs))
	for path, a := range aggs {
		result[path] = &model.GitFileMetrics{
			FilePath:      path,
			LinesAdded:    a.added,
			LinesDeleted:  a.deleted,
			Commits:       a.commits,
			BugfixCommits: a.bugfixCommits,
			Authors:       len(a.authors),
		}
	}
	return result, nil
}
