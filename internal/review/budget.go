package review

import (
	"sort"
	"strconv"
	"strings"

	"github.com/rejisterjack/scrute/internal/config"
	"github.com/rejisterjack/scrute/internal/gitdiff"
)

// Pack splits reviewable files into requests that fit maxBytes.
// Excluded paths are dropped. Binaries and submodules are listed and not sent.
// A single file larger than maxBytes is omitted.
func Pack(files []gitdiff.FileChange, patterns []string, contextLines, maxBytes int) (chunks [][]Section, omitted, binaries, submodules []string) {
	var sections []Section
	for _, file := range files {
		if config.Excluded(patterns, file.Path) || (file.OldPath != "" && config.Excluded(patterns, file.OldPath)) {
			continue
		}
		if file.Submodule {
			submodules = append(submodules, file.Path)
			continue
		}
		if file.Binary {
			binaries = append(binaries, file.Path)
			continue
		}
		body := sectionBody(file, contextLines)
		if len(body) > maxBytes {
			omitted = append(omitted, file.Path)
			continue
		}
		sections = append(sections, Section{Path: file.Path, Body: body})
	}

	var current []Section
	size := 0
	for _, section := range sections {
		if len(current) > 0 && size+len(section.Body)+1 > maxBytes {
			chunks = append(chunks, current)
			current = nil
			size = 0
		}
		current = append(current, section)
		size += len(section.Body) + 1
	}
	if len(current) > 0 {
		chunks = append(chunks, current)
	}
	return chunks, omitted, binaries, submodules
}

// Merge combines chunk reviews and drops duplicate issues.
func Merge(parts []Review) Review {
	var summaries []string
	var issues []Issue
	seen := map[string]bool{}
	for _, part := range parts {
		if text := strings.TrimSpace(part.Summary); text != "" {
			summaries = append(summaries, text)
		}
		for _, issue := range part.Issues {
			key := issue.Severity + "\x00" + issue.File + "\x00" + strconv.Itoa(issue.Line) + "\x00" + issue.Title
			if seen[key] {
				continue
			}
			seen[key] = true
			issues = append(issues, issue)
		}
	}
	sort.SliceStable(issues, func(i, j int) bool {
		if Rank(issues[i].Severity) != Rank(issues[j].Severity) {
			return Rank(issues[i].Severity) < Rank(issues[j].Severity)
		}
		if issues[i].File != issues[j].File {
			return issues[i].File < issues[j].File
		}
		if issues[i].Line != issues[j].Line {
			return issues[i].Line < issues[j].Line
		}
		return issues[i].Title < issues[j].Title
	})
	if issues == nil {
		issues = []Issue{}
	}
	return Review{Summary: strings.Join(summaries, "\n\n"), Issues: issues}
}
