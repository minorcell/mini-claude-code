package tui

import (
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
	"unicode"

	tea "github.com/charmbracelet/bubbletea"
)

type fileCandidate struct {
	Path  string
	Base  string
	Dir   string
	Score int
}

type filePickerState struct {
	Active   bool
	Loading  bool
	Query    string
	Start    int
	End      int
	Results  []fileCandidate
	Selected int
}

type fileIndexLoadedMsg struct {
	files []string
	err   error
}

func buildFileIndexCmd(root string) tea.Cmd {
	return func() tea.Msg {
		files, err := collectWorkspaceFiles(root)
		return fileIndexLoadedMsg{
			files: files,
			err:   err,
		}
	}
}

func collectWorkspaceFiles(root string) ([]string, error) {
	files := make([]string, 0, 256)
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			if entry != nil && entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if path == root {
			return nil
		}
		if entry.IsDir() {
			if shouldSkipCandidateDir(entry.Name()) {
				return filepath.SkipDir
			}
			return nil
		}

		rel, err := filepath.Rel(root, path)
		if err != nil {
			return nil
		}
		files = append(files, filepath.ToSlash(rel))
		return nil
	})
	if err != nil {
		return nil, err
	}

	sort.Strings(files)
	return files, nil
}

func shouldSkipCandidateDir(name string) bool {
	switch name {
	case ".git", "node_modules", "vendor", ".next", "dist", "build", ".cache", "coverage", "tmp":
		return true
	default:
		return false
	}
}

func detectFilePickerContext(value string, cursor int) filePickerState {
	runes := []rune(value)
	if cursor < 0 {
		cursor = 0
	}
	if cursor > len(runes) {
		cursor = len(runes)
	}

	for i := cursor - 1; i >= 0; i-- {
		if isFileCandidateTerminator(runes[i]) {
			break
		}
		if runes[i] != '@' {
			continue
		}
		if i > 0 && !isFileCandidateBoundary(runes[i-1]) {
			return filePickerState{}
		}

		end := cursor
		for end < len(runes) && !isFileCandidateTerminator(runes[end]) {
			end++
		}

		return filePickerState{
			Active: true,
			Query:  strings.TrimSpace(string(runes[i+1 : cursor])),
			Start:  i,
			End:    end,
		}
	}

	return filePickerState{}
}

func isFileCandidateBoundary(r rune) bool {
	return unicode.IsSpace(r) || strings.ContainsRune("([{\"'`", r)
}

func isFileCandidateTerminator(r rune) bool {
	if unicode.IsSpace(r) {
		return true
	}
	return strings.ContainsRune(",;:()[]{}<>\"'`", r)
}

func searchFileCandidates(files []string, query string, limit int) []fileCandidate {
	normalized := strings.ToLower(strings.TrimSpace(query))
	if limit <= 0 {
		limit = 8
	}

	results := make([]fileCandidate, 0, min(limit, len(files)))
	for _, path := range files {
		candidate, ok := scoreFileCandidate(path, normalized)
		if !ok {
			continue
		}
		results = append(results, candidate)
	}

	sort.Slice(results, func(i int, j int) bool {
		if results[i].Score != results[j].Score {
			return results[i].Score < results[j].Score
		}
		if len(results[i].Path) != len(results[j].Path) {
			return len(results[i].Path) < len(results[j].Path)
		}
		return results[i].Path < results[j].Path
	})

	if len(results) > limit {
		results = results[:limit]
	}
	return results
}

func scoreFileCandidate(path string, query string) (fileCandidate, bool) {
	base := strings.ToLower(filepath.Base(path))
	full := strings.ToLower(path)
	var score int

	switch {
	case query == "":
		score = pathDepth(path)*10 + len(path)
	case strings.HasPrefix(base, query):
		score = 0 + len(base) - len(query)
	case strings.HasPrefix(full, query):
		score = 20 + len(full) - len(query)
	case strings.Contains(base, query):
		score = 40 + strings.Index(base, query)
	case strings.Contains(full, query):
		score = 80 + strings.Index(full, query)
	default:
		baseScore, baseOK := subsequenceScore(base, query)
		fullScore, fullOK := subsequenceScore(full, query)
		switch {
		case baseOK:
			score = 140 + baseScore
		case fullOK:
			score = 220 + fullScore
		default:
			return fileCandidate{}, false
		}
	}

	dir := filepath.Dir(path)
	if dir == "." {
		dir = ""
	}
	return fileCandidate{
		Path:  path,
		Base:  filepath.Base(path),
		Dir:   dir,
		Score: score,
	}, true
}

func subsequenceScore(value string, query string) (int, bool) {
	if query == "" {
		return len(value), true
	}

	score := 0
	lastIndex := -1
	for _, q := range query {
		index := strings.IndexRune(value[lastIndex+1:], q)
		if index < 0 {
			return 0, false
		}
		absolute := lastIndex + 1 + index
		if lastIndex >= 0 {
			score += absolute - lastIndex - 1
		}
		lastIndex = absolute
	}
	score += len(value) - len(query)
	return score, true
}

func pathDepth(path string) int {
	if path == "" {
		return 0
	}
	return strings.Count(path, "/")
}

func min(a int, b int) int {
	if a < b {
		return a
	}
	return b
}
