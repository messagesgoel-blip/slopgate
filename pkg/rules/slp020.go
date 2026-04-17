package rules

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/messagesgoel-blip/slopgate/pkg/diff"
)

// SLP020 flags use of insecure random number generators and weak
// hash functions. AI agents frequently use non-cryptographic PRNGs
// and weak hashes when security context demands strong ones.
//
// Two tiers:
//   - warn: when security-context keywords appear nearby (password,
//     token, secret, key, session, nonce, salt, credential, auth)
//   - info: otherwise
//
// Patterns flagged:
//   - Insecure random: math/rand import (Go), random. (Python, not secrets.),
//     Math.random() (JS), java.util.Random (Java)
//   - Insecure hash: md5/sha1 (Go, Python, JS, Java)
//
// Exempt: test files; doc files; Python secrets module; Go crypto/rand.
type SLP020 struct{}

func (SLP020) ID() string                { return "SLP020" }
func (SLP020) DefaultSeverity() Severity { return SeverityInfo }
func (SLP020) Description() string {
	return "insecure random or weak hash — use cryptographically secure alternative"
}

var slp020Patterns = []struct {
	re       *regexp.Regexp
	lang     string
	category string
	example  string
}{
	// Insecure random
	{regexp.MustCompile(`math/rand`), "go", "random", "math/rand"},
	{regexp.MustCompile(`rand\.\w+\(`), "go", "random", "rand.*() (with math/rand)"},
	{regexp.MustCompile(`random\.`), "py", "random", "random."},
	{regexp.MustCompile(`Math\.random\s*\(`), "js", "random", "Math.random()"},
	{regexp.MustCompile(`java\.util\.Random`), "java", "random", "java.util.Random"},
	// Insecure hash
	{regexp.MustCompile(`md5\.(Sum|New)\b|sha1\.(Sum|New)\b`), "go", "hash", "crypto/md5 or crypto/sha1"},
	{regexp.MustCompile(`hashlib\.(md5|sha1)\b`), "py", "hash", "hashlib.md5/sha1"},
	{regexp.MustCompile(`createHash\s*\(\s*(?:'md5'|'sha1'|"md5"|"sha1")\s*\)`), "js", "hash", "createHash('md5'/'sha1')"},
	{regexp.MustCompile(`(?:MD5|SHA-1|SHA1)\b`), "java", "hash", "MD5/SHA-1"},
}

var slp020SecContext = regexp.MustCompile(`(?i)(?:password|token|secret|key|session|nonce|salt|credential|auth)`)

// slp020PythonSecure matches Python secrets module usage (safe random).
var slp020PythonSecure = regexp.MustCompile(`secrets\.`)

// slp020GoSecureRand detects crypto/rand import or usage (secure random).
var slp020GoSecureRand = regexp.MustCompile(`crypto/rand`)

// slp020MathRand detects math/rand import (insecure random).
var slp020MathRand = regexp.MustCompile(`math/rand`)

func slp020FileLang(path string) string {
	if isGoFile(path) {
		return "go"
	}
	if isPythonFile(path) {
		return "py"
	}
	if isJSOrTSFile(path) {
		return "js"
	}
	if isJavaFile(path) {
		return "java"
	}
	return ""
}

func (r SLP020) Check(d *diff.Diff) []Finding {
	var out []Finding
	for _, f := range d.Files {
		if f.IsDelete || isDocFile(f.Path) {
			continue
		}
		lang := slp020FileLang(f.Path)
		if lang == "" {
			continue
		}
		isTest := isGoTestFile(f.Path) || isJavaTestFile(f.Path) ||
			isPythonTestFile(f.Path) || isJSTestFile(f.Path) || isRustTestFile(f.Path)
		if isTest {
			continue
		}

		// Go: check if the file imports crypto/rand or math/rand anywhere in the diff.
		goUsesCryptoRand := false
		goUsesMathRand := false
		if lang == "go" {
			for _, h := range f.Hunks {
				for _, ln := range h.Lines {
					if slp020GoSecureRand.MatchString(ln.Content) {
						goUsesCryptoRand = true
					}
					if slp020MathRand.MatchString(ln.Content) {
						goUsesMathRand = true
					}
				}
			}
		}

		for _, ln := range f.AddedLines() {
			raw := ln.Content
			trimmed := strings.TrimLeft(raw, " \t")
			if strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "#") {
				continue
			}

			// For Go, use raw content (import paths are in strings).
			// For JS/Java, use raw content (hash algorithm names are in strings).
			// For Python, use stripped content.
			checkAgainst := raw
			if lang == "py" {
				checkAgainst = stripCommentAndStrings(raw)
			}
			if checkAgainst == "" {
				continue
			}

			// Python: skip lines using secrets module (cryptographically secure).
			if lang == "py" && slp020PythonSecure.MatchString(checkAgainst) {
				continue
			}

			for _, p := range slp020Patterns {
				if p.lang != "" && lang != p.lang {
					continue
				}
				// Go: skip random-category match if file imports crypto/rand.
				if lang == "go" && goUsesCryptoRand && p.category == "random" {
					continue
				}
				// Go: skip rand.*() call-site pattern without math/rand import (ambiguous).
				if lang == "go" && p.example == "rand.*() (with math/rand)" && !goUsesMathRand {
					continue
				}
				if p.re.MatchString(checkAgainst) {
					sev := SeverityInfo
					if slp020SecContext.MatchString(raw) {
						sev = SeverityWarn
					}
					out = append(out, Finding{
						RuleID:   r.ID(),
						Severity: sev,
						File:     f.Path,
						Line:     ln.NewLineNo,
						Message:  fmt.Sprintf("insecure %s — use cryptographic alternative instead of %s", p.category, p.example),
						Snippet:  strings.TrimSpace(ln.Content),
					})
					break
				}
			}
		}
	}
	return out
}
