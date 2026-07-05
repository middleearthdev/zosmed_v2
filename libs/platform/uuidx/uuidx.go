// Package uuidx converts between hyphenated UUID strings and pgtype.UUID.
// It lives in the neutral platform layer so transport code (apps/api, apps/worker)
// and kits share ONE implementation (§12a-1 DRY) instead of importing a kit for a
// generic concern (§5a boundary — M5).
package uuidx

import (
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgtype"
)

// Parse parses a hyphenated UUID string (xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx)
// into a pgtype.UUID. Hyphens are optional; any other input is rejected.
func Parse(s string) (pgtype.UUID, error) {
	clean := strings.ReplaceAll(s, "-", "")
	b, err := hex.DecodeString(clean)
	if err != nil || len(b) != 16 {
		return pgtype.UUID{}, fmt.Errorf("uuidx: invalid UUID %q", s)
	}
	var arr [16]byte
	copy(arr[:], b)
	return pgtype.UUID{Bytes: arr, Valid: true}, nil
}

// Format formats a pgtype.UUID as a lowercase hyphenated UUID string.
func Format(u pgtype.UUID) string {
	b := u.Bytes
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
