package rules

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/messagesgoel-blip/slopgate/pkg/diff"
)

// SLP128 flags interactive bot jobs that are enqueued with a positive BullMQ
// priority. BullMQ treats lower numeric values as higher priority, so
// priority: 1 can accidentally delay user-facing jobs behind default jobs.
type SLP128 struct{}

func (SLP128) ID() string                { return "SLP128" }
func (SLP128) DefaultSeverity() Severity { return SeverityWarn }
func (SLP128) Description() string {
	return "interactive bot queue job uses positive BullMQ priority"
}

// SLP129 flags live secrets/config committed in tracked .env files.
type SLP129 struct{}

func (SLP129) ID() string                { return "SLP129" }
func (SLP129) DefaultSeverity() Severity { return SeverityBlock }
func (SLP129) Description() string {
	return "tracked .env file contains live-looking secret or service binding"
}

// SLP130 flags production-origin navigation hardcoded into app code.
type SLP130 struct{}

func (SLP130) ID() string                { return "SLP130" }
func (SLP130) DefaultSeverity() Severity { return SeverityWarn }
func (SLP130) Description() string {
	return "hardcoded external origin in browser navigation"
}

// SLP131 flags nested React links/anchors, which produce invalid interactive
// markup and can break routing/accessibility.
type SLP131 struct{}

func (SLP131) ID() string                { return "SLP131" }
func (SLP131) DefaultSeverity() Severity { return SeverityWarn }
func (SLP131) Description() string {
	return "nested Link/anchor elements create invalid interactive markup"
}

// SLP132 flags global keyboard shortcuts that do not guard editable targets.
type SLP132 struct{}

func (SLP132) ID() string                { return "SLP132" }
func (SLP132) DefaultSeverity() Severity { return SeverityWarn }
func (SLP132) Description() string {
	return "global keyboard shortcut does not ignore editable controls"
}

// SLP133 flags Express router-level body parsers that commonly duplicate the
// app-level parser used for signature verification routes.
type SLP133 struct{}

func (SLP133) ID() string                { return "SLP133" }
func (SLP133) DefaultSeverity() Severity { return SeverityWarn }
func (SLP133) Description() string {
	return "Express router attaches body parser inline; verify it is not duplicated at app mount"
}

// SLP134 flags full transfer/failure arrays persisted into metadata or audit
// rows instead of bounded summaries.
type SLP134 struct{}

func (SLP134) ID() string                { return "SLP134" }
func (SLP134) DefaultSeverity() Severity { return SeverityWarn }
func (SLP134) Description() string {
	return "runtime metadata persists full transfer arrays instead of bounded summaries"
}

// SLP135 flags raw provider error messages persisted into summaries/audits.
type SLP135 struct{}

func (SLP135) ID() string                { return "SLP135" }
func (SLP135) DefaultSeverity() Severity { return SeverityWarn }
func (SLP135) Description() string {
	return "raw err.message persisted into user-visible summary or audit metadata"
}

var (
	slp128PositivePriority = regexp.MustCompile(`\bpriority\s*:\s*[1-9]\d*\b`)
	slp129EnvAssign        = regexp.MustCompile(`^\s*([A-Z0-9_]*(?:KEY|TOKEN|SECRET|PASSWORD|SUPABASE|ANON|URL)[A-Z0-9_]*)\s*=\s*(.+?)\s*$`)
	slp130Navigation       = regexp.MustCompile(`\b(?:window\.)?(?:location\.(?:assign|replace)|window\.open)\s*\(\s*["'\x60]https?://|\b(?:window\.)?location\.href\s*=\s*["'\x60]https?://`)
	slp133InlineParser     = regexp.MustCompile(`\bexpress\.(?:raw|json|urlencoded)\s*\(`)
	slp134ArrayField       = regexp.MustCompile(`\b(?:transferIds|skippedTransfers|deleteFailures)\s*:`)
	slp135RawErrMessage    = regexp.MustCompile(`\b(?:error|message)\s*:\s*(?:err|error|e)\.message\b`)
)

func (r SLP128) Check(d *diff.Diff) []Finding {
	var out []Finding
	for _, f := range d.Files {
		if f.IsDelete || isDocFile(f.Path) || isTestFile(f.Path) || !isJSOrTSFile(f.Path) {
			continue
		}
		for _, h := range f.Hunks {
			for i, ln := range h.Lines {
				if ln.Kind != diff.LineAdd {
					continue
				}
				if !slp128PositivePriority.MatchString(ln.Content) {
					continue
				}
				window := slpWindowText(h.Lines, i, 8, 8)
				if !strings.Contains(window, "queue.add") {
					continue
				}
				if !strings.Contains(window, "'bot.") && !strings.Contains(window, `"bot.`) &&
					!strings.Contains(window, "`bot.") && !strings.Contains(window, "buildJobEnvelope") {
					continue
				}
				out = append(out, Finding{
					RuleID:   r.ID(),
					Severity: r.DefaultSeverity(),
					File:     f.Path,
					Line:     ln.NewLineNo,
					Message:  "interactive bot job uses positive BullMQ priority; remove it or use a higher-urgency value consistently",
					Snippet:  strings.TrimSpace(ln.Content),
				})
			}
		}
	}
	return out
}

func (r SLP129) Check(d *diff.Diff) []Finding {
	var out []Finding
	for _, f := range d.Files {
		if f.IsDelete || !slp129IsTrackedEnvFile(f.Path) {
			continue
		}
		for _, ln := range f.AddedLines() {
			match := slp129EnvAssign.FindStringSubmatch(ln.Content)
			if match == nil {
				continue
			}
			value := strings.Trim(strings.TrimSpace(match[2]), `"'`)
			if slp129LooksPlaceholder(value) {
				continue
			}
			out = append(out, Finding{
				RuleID:   r.ID(),
				Severity: r.DefaultSeverity(),
				File:     f.Path,
				Line:     ln.NewLineNo,
				Message:  fmt.Sprintf("%s in tracked .env looks live; move it to secrets and commit only placeholders", match[1]),
				Snippet:  strings.TrimSpace(ln.Content),
			})
		}
	}
	return out
}

func (r SLP130) Check(d *diff.Diff) []Finding {
	var out []Finding
	for _, f := range d.Files {
		if f.IsDelete || isDocFile(f.Path) || isTestFile(f.Path) || !isJSOrTSFile(f.Path) {
			continue
		}
		for _, ln := range f.AddedLines() {
			if !slp130Navigation.MatchString(ln.Content) {
				continue
			}
			lower := strings.ToLower(ln.Content)
			if strings.Contains(lower, "localhost") || strings.Contains(lower, "127.0.0.1") {
				continue
			}
			out = append(out, Finding{
				RuleID:   r.ID(),
				Severity: r.DefaultSeverity(),
				File:     f.Path,
				Line:     ln.NewLineNo,
				Message:  "hardcoded production navigation breaks local/staging; derive origin from config or router state",
				Snippet:  strings.TrimSpace(ln.Content),
			})
		}
	}
	return out
}

func (r SLP131) Check(d *diff.Diff) []Finding {
	var out []Finding
	for _, f := range d.Files {
		if f.IsDelete || isDocFile(f.Path) || isTestFile(f.Path) || !strings.HasSuffix(strings.ToLower(f.Path), ".tsx") && !strings.HasSuffix(strings.ToLower(f.Path), ".jsx") {
			continue
		}
		for _, h := range f.Hunks {
			linkDepth := 0
			anchorDepth := 0
			for _, ln := range h.Lines {
				if ln.Kind == diff.LineDelete {
					continue
				}
				line := ln.Content
				openLink := slpHasOpeningTag(line, "Link")
				openAnchor := slpHasOpeningTag(line, "a")
				if ln.Kind == diff.LineAdd && (linkDepth > 0 && (openLink || openAnchor) || anchorDepth > 0 && openAnchor) {
					out = append(out, Finding{
						RuleID:   r.ID(),
						Severity: r.DefaultSeverity(),
						File:     f.Path,
						Line:     ln.NewLineNo,
						Message:  "nested Link/anchor detected; use a non-anchor wrapper or move child link outside",
						Snippet:  strings.TrimSpace(ln.Content),
					})
				}
				if openLink && !strings.Contains(line, "/>") {
					linkDepth++
				}
				if openAnchor && !strings.Contains(line, "/>") {
					anchorDepth++
				}
				linkDepth -= strings.Count(line, "</Link")
				anchorDepth -= strings.Count(line, "</a>")
				if linkDepth < 0 {
					linkDepth = 0
				}
				if anchorDepth < 0 {
					anchorDepth = 0
				}
			}
		}
	}
	return out
}

func (r SLP132) Check(d *diff.Diff) []Finding {
	var out []Finding
	for _, f := range d.Files {
		if f.IsDelete || isDocFile(f.Path) || isTestFile(f.Path) || !isJSOrTSFile(f.Path) {
			continue
		}
		for _, h := range f.Hunks {
			for i, ln := range h.Lines {
				if ln.Kind != diff.LineAdd {
					continue
				}
				if !slp132ShortcutEntryLine(ln.Content) {
					continue
				}
				window := slpWindowText(h.Lines, i, 12, 20)
				if !slp132LooksLikeShortcut(window) || slp132HasEditableGuard(window) {
					continue
				}
				out = append(out, Finding{
					RuleID:   r.ID(),
					Severity: r.DefaultSeverity(),
					File:     f.Path,
					Line:     ln.NewLineNo,
					Message:  "global keyboard shortcut lacks editable-target guard for input/textarea/contenteditable",
					Snippet:  strings.TrimSpace(ln.Content),
				})
			}
		}
	}
	return out
}

func (r SLP133) Check(d *diff.Diff) []Finding {
	var out []Finding
	for _, f := range d.Files {
		if f.IsDelete || isDocFile(f.Path) || isTestFile(f.Path) || !isJSOrTSFile(f.Path) {
			continue
		}
		for _, h := range f.Hunks {
			for i, ln := range h.Lines {
				if ln.Kind != diff.LineAdd || !slp133InlineParser.MatchString(ln.Content) {
					continue
				}
				window := slpWindowText(h.Lines, i, 4, 4)
				if !strings.Contains(window, "router.") {
					continue
				}
				out = append(out, Finding{
					RuleID:   r.ID(),
					Severity: r.DefaultSeverity(),
					File:     f.Path,
					Line:     ln.NewLineNo,
					Message:  "router-level Express body parser may duplicate app-level parser; configure parsing in one layer",
					Snippet:  strings.TrimSpace(ln.Content),
				})
			}
		}
	}
	return out
}

func (r SLP134) Check(d *diff.Diff) []Finding {
	var out []Finding
	for _, f := range d.Files {
		if f.IsDelete || isDocFile(f.Path) || isTestFile(f.Path) || !isJSOrTSFile(f.Path) {
			continue
		}
		for _, h := range f.Hunks {
			for i, ln := range h.Lines {
				if ln.Kind != diff.LineAdd {
					continue
				}
				window := slpWindowText(h.Lines, i, 6, 6)
				if slp134ArrayField.MatchString(ln.Content) && strings.Contains(ln.Content, "summary.") {
					out = append(out, slp134Finding(r, f.Path, ln))
					continue
				}
				if strings.Contains(ln.Content, "JSON.stringify(summary)") && slp134MetadataContext(window) {
					out = append(out, slp134Finding(r, f.Path, ln))
				}
			}
		}
	}
	return out
}

func (r SLP135) Check(d *diff.Diff) []Finding {
	var out []Finding
	for _, f := range d.Files {
		if f.IsDelete || isDocFile(f.Path) || isTestFile(f.Path) || !isJSOrTSFile(f.Path) {
			continue
		}
		for _, h := range f.Hunks {
			for i, ln := range h.Lines {
				if ln.Kind != diff.LineAdd || !slp135RawErrMessage.MatchString(ln.Content) {
					continue
				}
				window := slpWindowText(h.Lines, i, 6, 6)
				if !slp135PersistenceContext(window) {
					continue
				}
				out = append(out, Finding{
					RuleID:   r.ID(),
					Severity: r.DefaultSeverity(),
					File:     f.Path,
					Line:     ln.NewLineNo,
					Message:  "raw err.message is persisted; store a sanitized code or bounded non-sensitive message",
					Snippet:  strings.TrimSpace(ln.Content),
				})
			}
		}
	}
	return out
}

func slpWindowText(lines []diff.Line, center, before, after int) string {
	start := center - before
	if start < 0 {
		start = 0
	}
	end := center + after
	if end >= len(lines) {
		end = len(lines) - 1
	}
	var b strings.Builder
	for i := start; i <= end; i++ {
		if lines[i].Kind == diff.LineDelete {
			continue
		}
		b.WriteString(lines[i].Content)
		b.WriteByte('\n')
	}
	return b.String()
}

func slp129IsTrackedEnvFile(path string) bool {
	lower := strings.ToLower(path)
	base := lower
	if idx := strings.LastIndex(base, "/"); idx >= 0 {
		base = base[idx+1:]
	}
	if strings.Contains(base, "example") || strings.Contains(base, "sample") ||
		strings.Contains(base, "template") {
		return false
	}
	return base == ".env" || strings.HasPrefix(base, ".env.") || strings.HasSuffix(base, ".env")
}

func slp129LooksPlaceholder(value string) bool {
	lower := strings.ToLower(strings.TrimSpace(value))
	if lower == "" || strings.HasPrefix(lower, "${") || strings.Contains(lower, "localhost") ||
		strings.Contains(lower, "127.0.0.1") {
		return true
	}
	placeholders := []string{"example", "placeholder", "changeme", "change_me", "your_", "test_", "mock", "<", ">"}
	for _, p := range placeholders {
		if strings.Contains(lower, p) {
			return true
		}
	}
	return false
}

func slpHasOpeningTag(line, tag string) bool {
	needle := "<" + tag
	return strings.Contains(line, needle+" ") || strings.Contains(line, needle+">") ||
		strings.Contains(line, needle+"\n") || strings.Contains(line, needle+"\t")
}

func slp132ShortcutEntryLine(line string) bool {
	lower := strings.ToLower(line)
	return strings.Contains(lower, "keydown") || strings.Contains(line, "handleKeyDown") ||
		strings.Contains(line, "onKeyDown")
}

func slp132LooksLikeShortcut(window string) bool {
	lower := strings.ToLower(window)
	return (strings.Contains(window, "metaKey") || strings.Contains(window, "ctrlKey") ||
		strings.Contains(lower, "escape") || strings.Contains(lower, "event.key") ||
		strings.Contains(lower, ".key")) &&
		(strings.Contains(window, "addEventListener") || strings.Contains(window, "onKeyDown") ||
			strings.Contains(window, "handleKeyDown"))
}

func slp132HasEditableGuard(window string) bool {
	lower := strings.ToLower(window)
	guards := []string{"activeelement", "event.target", "e.target", "target.tagname", "iscontenteditable", "closest(", "input", "textarea", "select", "contenteditable"}
	for _, guard := range guards {
		if strings.Contains(lower, guard) {
			return true
		}
	}
	return false
}

func slp134Finding(r SLP134, path string, ln diff.Line) Finding {
	return Finding{
		RuleID:   r.ID(),
		Severity: r.DefaultSeverity(),
		File:     path,
		Line:     ln.NewLineNo,
		Message:  "full transfer/failure arrays persisted into runtime metadata; store counts and capped samples",
		Snippet:  strings.TrimSpace(ln.Content),
	}
}

func slp134MetadataContext(window string) bool {
	lower := strings.ToLower(window)
	return strings.Contains(lower, "metadata") || strings.Contains(lower, "audit_logs") ||
		strings.Contains(lower, "lastrun") || strings.Contains(lower, "recentruns")
}

func slp135PersistenceContext(window string) bool {
	lower := strings.ToLower(window)
	terms := []string{"summary", "audit", "metadata", "failure", "failures", "deletefailures", "lastrun", "recentruns"}
	for _, term := range terms {
		if strings.Contains(lower, term) {
			return true
		}
	}
	return false
}
