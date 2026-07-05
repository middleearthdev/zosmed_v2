package uuidx_test

import (
	"testing"

	"github.com/zosmed/zosmed/libs/platform/uuidx"
)

func TestParse_RoundTrip(t *testing.T) {
	original := "aaaabbbb-cccc-dddd-eeee-ffffffffffff"
	u, err := uuidx.Parse(original)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if back := uuidx.Format(u); back != original {
		t.Errorf("Format(Parse(%q)) = %q, want original", original, back)
	}
}

func TestParse_InvalidInput(t *testing.T) {
	for _, s := range []string{"", "not-a-uuid", "gggggggg-0000-0000-0000-000000000000"} {
		if _, err := uuidx.Parse(s); err == nil {
			t.Errorf("Parse(%q) expected error, got nil", s)
		}
	}
}
