package seller_test

import (
	"testing"

	"github.com/zosmed/zosmed/libs/kits/seller"
)

func TestDetectKeepCode(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		wantCode string
		wantOK   bool
	}{
		// ── positive cases ───────────────────────────────────────────────────
		{name: "keep lowercase", text: "keep", wantCode: "keep", wantOK: true},
		{name: "keep uppercase", text: "KEEP", wantCode: "keep", wantOK: true},
		{name: "keep mixed", text: "Keep", wantCode: "keep", wantOK: true},
		{name: "keep with context", text: "keep dong kak!", wantCode: "keep", wantOK: true},
		{name: "keep leading whitespace", text: "  keep  ", wantCode: "keep", wantOK: true},

		{name: "order lowercase", text: "order", wantCode: "order", wantOK: true},
		{name: "order uppercase", text: "ORDER satu", wantCode: "order", wantOK: true},

		{name: "C standalone", text: "C", wantCode: "C", wantOK: true},
		{name: "C lowercase", text: "c", wantCode: "C", wantOK: true},
		{name: "C1", text: "C1", wantCode: "C1", wantOK: true},
		{name: "C1 lowercase", text: "c1", wantCode: "C1", wantOK: true},
		{name: "C3", text: "c3", wantCode: "C3", wantOK: true},
		{name: "C5 not in default list but matched by regex", text: "C5", wantCode: "C5", wantOK: true},
		{name: "C with context", text: "mau c1 kak", wantCode: "C1", wantOK: true},
		{name: "C1 trailing comment", text: "c1 dong, ada stok?", wantCode: "C1", wantOK: true},

		// ── negative cases — must NOT match ─────────────────────────────────
		{name: "coba — c prefix inside word", text: "coba dulu", wantOK: false},
		{name: "COD — 'c' inside word", text: "bisa COD?", wantOK: false},
		{name: "cek — 'c' inside word", text: "cek ongkir", wantOK: false},
		{name: "cc1 — double-c prefix", text: "cc1", wantOK: false},
		{name: "empty string", text: "", wantOK: false},
		{name: "random comment no code", text: "kak ada diskon ga?", wantOK: false},
		{name: "harga berapa", text: "harga berapa kak", wantOK: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			code, ok := seller.DetectKeepCode(tc.text)
			if ok != tc.wantOK {
				t.Errorf("DetectKeepCode(%q) ok=%v, want %v", tc.text, ok, tc.wantOK)
			}
			if ok && code != tc.wantCode {
				t.Errorf("DetectKeepCode(%q) code=%q, want %q", tc.text, code, tc.wantCode)
			}
		})
	}
}
