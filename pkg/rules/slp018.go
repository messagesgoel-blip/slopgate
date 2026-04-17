package rules

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/messagesgoel-blip/slopgate/pkg/diff"
)

// SLP018 flags overly broad exception catch clauses. AI agents default
// to the broadest catch to "handle all cases" instead of catching
// specific exception types. This masks bugs by swallowing unexpected
// errors that should propagate.
//
// Java patterns: catch (Exception e), catch (Throwable t), catch (RuntimeException e)
// Python patterns: except:, except Exception:, except BaseException:
//
// Exempt: test files.
type SLP018 struct{}

func (SLP018) ID() string                { return "SLP018" }
func (SLP018) DefaultSeverity() Severity { return SeverityWarn }
func (SLP018) Description() string {
	return "overly broad catch/except catches base exception type instead of a specific one"
}

var slp018Patterns = []struct {
	re      *regexp.Regexp
	lang    string
	example string
}{
	// Java: catch (Exception e), catch (final Exception e), catch (Throwable t), catch (RuntimeException e)
	{regexp.MustCompile(`catch\s*\(\s*(?:final\s+)?(?:Exception|Throwable|RuntimeException)\b`), "java", "catch (Exception e)"},
	// Python: except:, except Exception:, except BaseException:, except Exception as e:
	{regexp.MustCompile(`except\s*:|except\s+(?:Base)?Exception\s*[:\s]`), "py", "except Exception:"},
}

func (r SLP018) Check(d *diff.Diff) []Finding {
	var out []Finding
	for _, f := range d.Files {
		if f.IsDelete || isDocFile(f.Path) {
			continue
		}
		if isJavaFile(f.Path) && isJavaTestFile(f.Path) {
			continue
		}
		if isPythonFile(f.Path) && isPythonTestFile(f.Path) {
			continue
		}
		lang := slp018FileLang(f.Path)
		if lang == "" {
			continue
		}
		for _, ln := range f.AddedLines() {
			clean := stripCommentAndStrings(ln.Content)
			for _, p := range slp018Patterns {
				if p.lang != "" && lang != p.lang {
					continue
				}
				if p.re.MatchString(clean) {
					out = append(out, Finding{
						RuleID:   r.ID(),
						Severity: r.DefaultSeverity(),
						File:     f.Path,
						Line:     ln.NewLineNo,
						Message:  fmt.Sprintf("overly broad exception handler — catch a specific exception type instead of %s", p.example),
						Snippet:  strings.TrimSpace(ln.Content),
					})
					break
				}
			}
		}
	}
	return out
}

func slp018FileLang(path string) string {
	if isJavaFile(path) {
		return "java"
	}
	if isPythonFile(path) {
		return "py"
	}
	return ""
}
