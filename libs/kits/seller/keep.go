package seller

import (
	"regexp"
	"strings"
)

// KitKeywords is the authoritative list of seller-kit order detection keywords.
// Single source of truth (§12a-1). Kept in sync with
// packages/types/src/constants.ts KIT_KEYWORDS.seller: keep, c, c1, c3, order.
// Never hardcode these strings elsewhere — import this slice instead.
var KitKeywords = []string{"keep", "c", "c1", "c3", "order"}

// keepCodeRe matches keep/order codes case-insensitively at word boundaries.
//
// Patterns:
//   - "keep"  — literal keyword
//   - "order" — literal keyword
//   - "c", "c1", "c3", "c5", … — letter C optionally followed by digits
//
// \b prevents matching C inside words like "coba", "COD", "cek", "cc1".
var keepCodeRe = regexp.MustCompile(`(?i)\b(keep|order|c\d*)\b`)

// DetectKeepCode scans text for a seller-kit keep/order code.
// Returns the normalised code and ok=true when a match is found:
//   - C-codes are uppercased  (e.g. "c1" → "C1", "C" → "C")
//   - "keep" and "order" are returned lowercase
//
// Returns ("", false) when no recognised code is present.
// Tolerates leading/trailing whitespace and mixed case.
// NOT matched: "coba", "COD", "cek" — word-boundary prevents false positives.
func DetectKeepCode(text string) (code string, ok bool) {
	m := keepCodeRe.FindString(strings.TrimSpace(text))
	if m == "" {
		return "", false
	}
	lower := strings.ToLower(m)
	switch lower {
	case "keep", "order":
		return lower, true
	default:
		// C-code: normalise to uppercase (C, C1, C3, C5, …).
		return strings.ToUpper(m), true
	}
}
