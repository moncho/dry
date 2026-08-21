package composecli

import "testing"

func TestParseConfigHashes(t *testing.T) {
	out := "cache 781cb76aba47c364664944274469779eda48920aaa0a3c103989dc0a3b6ec339\n" +
		"web bf6121f23746cbb8aa6e190dbdd370ed3928bc89066e458132ea9a62bcbc8edd\n"

	got := parseConfigHashes(out)
	if len(got) != 2 {
		t.Fatalf("expected 2 services, got %d: %v", len(got), got)
	}
	if got["web"] != "bf6121f23746cbb8aa6e190dbdd370ed3928bc89066e458132ea9a62bcbc8edd" {
		t.Fatalf("wrong hash for web: %q", got["web"])
	}
}

func TestParseConfigHashes_IgnoresJunkLines(t *testing.T) {
	got := parseConfigHashes("\nweb abc123\nmalformed\n  \n")
	if len(got) != 1 || got["web"] != "abc123" {
		t.Fatalf("expected only the well-formed line, got %v", got)
	}
}
