package seller_test

import (
	"net/url"
	"strings"
	"testing"

	"github.com/zosmed/zosmed/libs/kits/seller"
)

func TestBuildWaLink(t *testing.T) {
	tests := []struct {
		name    string
		phone   string
		nama    string
		kode    string
		produk  string
		wantPfx string // must start with this prefix
	}{
		{
			name:    "basic",
			phone:   "6281234567890",
			nama:    "Budi",
			kode:    "C1",
			produk:  "Baju Ungu M",
			wantPfx: "https://wa.me/6281234567890?text=",
		},
		{
			name:    "special chars in produk",
			phone:   "6281234567890",
			nama:    "Siti & Dewi",
			kode:    "keep",
			produk:  "Tas Kulit #5",
			wantPfx: "https://wa.me/6281234567890?text=",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			link := seller.BuildWaLink(tc.phone, tc.nama, tc.kode, tc.produk)

			if !strings.HasPrefix(link, tc.wantPfx) {
				t.Errorf("BuildWaLink prefix mismatch: got %q, want prefix %q", link, tc.wantPfx)
			}

			// The query param must be URL-encoded (no raw spaces).
			rawQuery := strings.TrimPrefix(link, "https://wa.me/"+tc.phone+"?")
			params, err := url.ParseQuery(rawQuery)
			if err != nil {
				t.Fatalf("BuildWaLink produced unparseable query: %v", err)
			}
			text := params.Get("text")
			if text == "" {
				t.Error("BuildWaLink: text query param is empty")
			}
			// {nama}, {kode}, {produk} must appear in the decoded text.
			if !strings.Contains(text, tc.nama) {
				t.Errorf("BuildWaLink text missing nama %q; got: %q", tc.nama, text)
			}
			if !strings.Contains(text, tc.kode) {
				t.Errorf("BuildWaLink text missing kode %q; got: %q", tc.kode, text)
			}
			if !strings.Contains(text, tc.produk) {
				t.Errorf("BuildWaLink text missing produk %q; got: %q", tc.produk, text)
			}
		})
	}
}

func TestBuildPrivateReplyText(t *testing.T) {
	t.Run("uses default template when tmpl empty", func(t *testing.T) {
		text := seller.BuildPrivateReplyText("", "Andi", "C1", "Kaos Polos L", "https://wa.me/x")
		if !strings.Contains(text, "Andi") {
			t.Error("expected nama in reply text")
		}
		if !strings.Contains(text, "C1") {
			t.Error("expected kode in reply text")
		}
		if !strings.Contains(text, "Kaos Polos L") {
			t.Error("expected produk in reply text")
		}
		if !strings.Contains(text, "https://wa.me/x") {
			t.Error("expected wa_link in reply text")
		}
	})

	t.Run("honours custom template", func(t *testing.T) {
		tmpl := "Order {kode} for {produk} confirmed, {nama}. Go: {wa_link}"
		text := seller.BuildPrivateReplyText(tmpl, "Sari", "keep", "Celana Jeans", "https://wa.me/y")
		want := "Order keep for Celana Jeans confirmed, Sari. Go: https://wa.me/y"
		if text != want {
			t.Errorf("BuildPrivateReplyText = %q, want %q", text, want)
		}
	})

	t.Run("whitespace-only tmpl falls back to default", func(t *testing.T) {
		text := seller.BuildPrivateReplyText("   ", "X", "C3", "P", "u")
		if !strings.Contains(text, "X") {
			t.Error("expected default template to be used")
		}
	})
}
