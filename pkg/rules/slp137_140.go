package rules

import (
	"io/fs"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/messagesgoel-blip/slopgate/pkg/diff"
)

// SLP137 flags explicit BullMQ bot priorities introduced while sibling
// call-sites in the repo still enqueue equivalent bot jobs with default
// priority. That mixture caused real CodeRabbit findings because BullMQ v5
// processes default-priority jobs ahead of positive priorities.
type SLP137 struct{}

func (SLP137) ID() string                { return "SLP137" }
func (SLP137) DefaultSeverity() Severity { return SeverityWarn }
func (SLP137) Description() string {
	return "bot queue uses mixed explicit/default BullMQ priority across sibling call sites"
}

// SLP138 flags provider operations that forward only token auth even though
// surrounding context shows credential-based auth is available too.
type SLP138 struct{}

func (SLP138) ID() string                { return "SLP138" }
func (SLP138) DefaultSeverity() Severity { return SeverityWarn }
func (SLP138) Description() string {
	return "provider call forwards token auth but drops available creds/credentials context"
}

// SLP139 flags partial S3 hardening rollouts: a helper is added in one path
// while sibling repo call-sites still parse raw S3 credential blobs and create
// S3 clients directly.
type SLP139 struct{}

func (SLP139) ID() string                { return "SLP139" }
func (SLP139) DefaultSeverity() Severity { return SeverityWarn }
func (SLP139) Description() string {
	return "S3 hardening helper added but sibling call sites still parse raw credential blobs"
}

// SLP140 flags hardening helpers that are applied to generic token variables
// without a provider-family or JSON-blob guard.
type SLP140 struct{}

func (SLP140) ID() string                { return "SLP140" }
func (SLP140) DefaultSeverity() Severity { return SeverityWarn }
func (SLP140) Description() string {
	return "credential hardening helper is called on generic token input without JSON/provider guard"
}

var (
	slp137PositivePriorityLine = regexp.MustCompile(`\bpriority\s*:\s*[1-9]\d*\b`)
	slp137QueueAdd             = regexp.MustCompile(`\bqueue\s*\.\s*add\s*\(`)
	slp137BotJobPattern        = regexp.MustCompile(`['"` + "`" + `]bot\.[^'"` + "`" + `]+['"` + "`" + `]|buildJobEnvelope\s*\(\s*['"` + "`" + `]bot\.`)

	slp138TokenField      = regexp.MustCompile(`\btoken\s*:\s*[^,}]+`)
	slp138CredsContext    = regexp.MustCompile(`\b(?:creds|credentials)\b|\.creds\b|\.credentials\b`)
	slp138AuthField       = regexp.MustCompile(`\b(?:creds|credentials|auth)\s*:`)
	slp138CreateFolder    = regexp.MustCompile(`\bcreateFolder\s*\(`)
	slp138CreateFolderMsg = "provider createFolder call forwards token auth but drops available creds/auth context"

	slp139HardeningHook = regexp.MustCompile(`\b(?:parseAndNormalizeStoredS3Creds|s3ClientConfigSafe|assertS3EndpointIsPublic|normalizeServerCredentials|hardenServerCredentials)\b`)
	slp139RawS3Client   = regexp.MustCompile(`new\s+S3Client\s*\(`)
	slp139RawCredParse  = regexp.MustCompile(`JSON\.parse\s*\(`)

	slp140HardenerCall  = regexp.MustCompile(`\b(?:await\s+)?(?:hardenServerCredentials|normalizeServerCredentials)\s*\(\s*provider\s*,\s*(?:accessToken|refreshToken|token)\b`)
	slp140ProviderGuard = regexp.MustCompile(`SERVER_PROVIDERS\.has\s*\(\s*provider\s*\)|\bprovider\s*===\s*['"` + "`" + `][^'"` + "`" + `]+['"` + "`" + `]|\bprovider\s*!==\s*['"` + "`" + `][^'"` + "`" + `]+['"` + "`" + `]`)
	slp140JSONGuard     = regexp.MustCompile(`JSON\.parse\s*\(\s*(?:accessToken|refreshToken|token)\s*\)|startsWith\s*\(\s*['"` + "`" + `]\{['"` + "`" + `]\s*\)|trim\(\)\.startsWith\s*\(\s*['"` + "`" + `]\{['"` + "`" + `]\s*\)|looksLikeJson|isJson`)
)

func slpRepoJSFiles(d *diff.Diff) ([]string, error) {
	if d == nil || d.RepoRoot == "" {
		return nil, nil
	}
	var out []string
	if err := filepath.WalkDir(d.RepoRoot, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			name := entry.Name()
			switch name {
			case ".git", "node_modules", "dist", "build", ".next", ".turbo", "coverage":
				return filepath.SkipDir
			}
			return nil
		}
		rel, err := filepath.Rel(d.RepoRoot, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if isJSOrTSFile(rel) && !isTestFile(rel) {
			out = append(out, rel)
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return out, nil
}

func slpReadRepoFile(d *diff.Diff, relPath string) (string, bool) {
	return slp007FileContent(d, relPath)
}

func slp137HasAddedPositivePriorityBotCalls(d *diff.Diff) bool {
	if d == nil {
		return false
	}
	for _, f := range d.Files {
		if f.IsDelete || isDocFile(f.Path) || isTestFile(f.Path) || !isJSOrTSFile(f.Path) {
			continue
		}
		for _, h := range f.Hunks {
			for i, ln := range h.Lines {
				if ln.Kind != diff.LineAdd || !slp137PositivePriorityLine.MatchString(ln.Content) {
					continue
				}
				window := slpWindowText(h.Lines, i, 8, 2)
				if slp137QueueAdd.MatchString(window) && slp137BotJobPattern.MatchString(window) {
					return true
				}
			}
		}
	}
	return false
}

func slp137HasUnprioritizedBotCalls(d *diff.Diff) bool {
	files, err := slpRepoJSFiles(d)
	if err != nil {
		return false
	}
	for _, relPath := range files {
		content, ok := slpReadRepoFile(d, relPath)
		if !ok {
			continue
		}
		lines := strings.Split(content, "\n")
		for i, line := range lines {
			if !slp137QueueAdd.MatchString(line) {
				continue
			}
			end := i + 6
			if end > len(lines) {
				end = len(lines)
			}
			window := strings.Join(lines[i:end], "\n")
			if slp137BotJobPattern.MatchString(window) && !slp137PositivePriorityLine.MatchString(window) {
				return true
			}
		}
	}
	return false
}

func (r SLP137) Check(d *diff.Diff) []Finding {
	if !slp137HasAddedPositivePriorityBotCalls(d) {
		return nil
	}
	if !slp137HasUnprioritizedBotCalls(d) {
		return nil
	}

	var out []Finding
	for _, f := range d.Files {
		if f.IsDelete || isDocFile(f.Path) || isTestFile(f.Path) || !isJSOrTSFile(f.Path) {
			continue
		}
		for _, h := range f.Hunks {
			for i, ln := range h.Lines {
				if ln.Kind != diff.LineAdd || !slp137PositivePriorityLine.MatchString(ln.Content) {
					continue
				}
				window := slpWindowText(h.Lines, i, 8, 2)
				if !slp137QueueAdd.MatchString(window) || !slp137BotJobPattern.MatchString(window) {
					continue
				}
				out = append(out, Finding{
					RuleID:   r.ID(),
					Severity: r.DefaultSeverity(),
					File:     f.Path,
					Line:     ln.NewLineNo,
					Message:  "positive BullMQ bot priority introduced while sibling repo call sites still use default priority",
					Snippet:  strings.TrimSpace(ln.Content),
				})
			}
		}
	}
	return out
}

func (r SLP138) Check(d *diff.Diff) []Finding {
	var out []Finding
	for _, f := range d.Files {
		if f.IsDelete || isDocFile(f.Path) || isTestFile(f.Path) || !isJSOrTSFile(f.Path) {
			continue
		}
		for _, h := range f.Hunks {
			for i, ln := range h.Lines {
				if ln.Kind != diff.LineAdd || !slp138TokenField.MatchString(ln.Content) {
					continue
				}
				window := slpWindowText(h.Lines, i, 8, 8)
				if !slp138CreateFolder.MatchString(window) {
					continue
				}
				if !slp138CredsContext.MatchString(window) || slp138AuthField.MatchString(window) {
					continue
				}
				out = append(out, Finding{
					RuleID:   r.ID(),
					Severity: r.DefaultSeverity(),
					File:     f.Path,
					Line:     ln.NewLineNo,
					Message:  slp138CreateFolderMsg,
					Snippet:  strings.TrimSpace(ln.Content),
				})
			}
		}
	}
	return out
}

func slp139HasRawS3Sibling(d *diff.Diff, excludePath string) bool {
	rawPaths := slp139RawS3Paths(d)
	return slp139HasRawS3SiblingIn(rawPaths, excludePath)
}

func slp139CandidatePaths(d *diff.Diff) map[string]bool {
	candidates := map[string]bool{}
	if d == nil {
		return candidates
	}
	for _, f := range d.Files {
		if f.IsDelete || isDocFile(f.Path) || isTestFile(f.Path) || !isJSOrTSFile(f.Path) {
			continue
		}
		for _, ln := range f.AddedLines() {
			if slp139HardeningHook.MatchString(ln.Content) {
				candidates[f.Path] = true
				break
			}
		}
	}
	return candidates
}

func slp139RawS3Paths(d *diff.Diff) map[string]bool {
	rawPaths := map[string]bool{}
	files, err := slpRepoJSFiles(d)
	if err != nil {
		return rawPaths
	}
	for _, relPath := range files {
		content, ok := slpReadRepoFile(d, relPath)
		if !ok {
			continue
		}
		if !slp139RawS3Client.MatchString(content) || !slp139RawCredParse.MatchString(content) {
			continue
		}
		rawPaths[relPath] = true
	}
	return rawPaths
}

func slp139HasRawS3SiblingIn(rawPaths map[string]bool, excludePath string) bool {
	for relPath := range rawPaths {
		if relPath != excludePath {
			return true
		}
	}
	return false
}

func (r SLP139) Check(d *diff.Diff) []Finding {
	candidates := slp139CandidatePaths(d)
	if len(candidates) == 0 {
		return nil
	}
	rawPaths := slp139RawS3Paths(d)
	if len(rawPaths) == 0 {
		return nil
	}

	var out []Finding
	for _, f := range d.Files {
		if f.IsDelete || isDocFile(f.Path) || isTestFile(f.Path) || !isJSOrTSFile(f.Path) {
			continue
		}
		if !candidates[f.Path] || !slp139HasRawS3SiblingIn(rawPaths, f.Path) {
			continue
		}
		for _, ln := range f.AddedLines() {
			if !slp139HardeningHook.MatchString(ln.Content) {
				continue
			}
			out = append(out, Finding{
				RuleID:   r.ID(),
				Severity: r.DefaultSeverity(),
				File:     f.Path,
				Line:     ln.NewLineNo,
				Message:  "S3 hardening helper added here, but sibling repo paths still parse raw credential blobs into S3Client directly",
				Snippet:  strings.TrimSpace(ln.Content),
			})
			break
		}
	}
	return out
}

func (r SLP140) Check(d *diff.Diff) []Finding {
	var out []Finding
	for _, f := range d.Files {
		if f.IsDelete || isDocFile(f.Path) || isTestFile(f.Path) || !isJSOrTSFile(f.Path) {
			continue
		}
		for _, h := range f.Hunks {
			for i, ln := range h.Lines {
				if ln.Kind != diff.LineAdd || !slp140HardenerCall.MatchString(ln.Content) {
					continue
				}
				window := slpWindowText(h.Lines, i, 6, 2)
				if slp140ProviderGuard.MatchString(window) || slp140JSONGuard.MatchString(window) {
					continue
				}
				out = append(out, Finding{
					RuleID:   r.ID(),
					Severity: r.DefaultSeverity(),
					File:     f.Path,
					Line:     ln.NewLineNo,
					Message:  "credential hardening helper is applied to a generic token without a provider-family or JSON guard",
					Snippet:  strings.TrimSpace(ln.Content),
				})
			}
		}
	}
	return out
}
