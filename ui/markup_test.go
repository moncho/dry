package ui

import (
	"strings"
	"testing"
)

func TestRenderMarkup_PlainText(t *testing.T) {
	result := RenderMarkup("hello world")
	if result != "hello world" {
		t.Errorf("expected plain text unchanged, got %q", result)
	}
}

func TestRenderMarkup_SingleTag(t *testing.T) {
	result := RenderMarkup("<white>hello</>")
	if !strings.Contains(result, "hello") {
		t.Errorf("expected result to contain 'hello', got %q", result)
	}
	// Should contain ANSI escape codes
	if result == "hello" {
		t.Error("expected ANSI styling, got plain text")
	}
}

func TestRenderMarkup_NestedTags(t *testing.T) {
	// <b>[H]:<darkgrey>Help</> should produce bold "[H]:" and bold+darkgrey "Help"
	result := RenderMarkup("<b>[H]:<darkgrey>Help</>")
	if !strings.Contains(result, "[H]:") {
		t.Errorf("expected result to contain '[H]:', got %q", result)
	}
	if !strings.Contains(result, "Help") {
		t.Errorf("expected result to contain 'Help', got %q", result)
	}
	// Should contain ANSI codes (not plain text)
	if result == "[H]:Help" {
		t.Error("expected ANSI styling, got plain text")
	}
}

func TestRenderMarkup_ResetBetweenGroups(t *testing.T) {
	result := RenderMarkup("<b>[H]:<darkgrey>Help</> <b>[Q]:<darkgrey>Quit</>")
	if !strings.Contains(result, "[H]:") {
		t.Errorf("expected '[H]:' in result")
	}
	if !strings.Contains(result, "Help") {
		t.Errorf("expected 'Help' in result")
	}
	if !strings.Contains(result, "[Q]:") {
		t.Errorf("expected '[Q]:' in result")
	}
}

func TestRenderMarkup_BlueSeparator(t *testing.T) {
	result := RenderMarkup("<blue>|</>")
	if !strings.Contains(result, "|") {
		t.Errorf("expected '|' in result, got %q", result)
	}
}

func TestTokenize(t *testing.T) {
	tokens := Tokenize("<b>[H]:<darkgrey>Help</>", SupportedTags)
	expected := []string{"<b>", "[H]:", "<darkgrey>", "Help", "</>"}
	if len(tokens) != len(expected) {
		t.Fatalf("expected %d tokens, got %d: %v", len(expected), len(tokens), tokens)
	}
	for i, tok := range tokens {
		if tok != expected[i] {
			t.Errorf("token %d: expected %q, got %q", i, expected[i], tok)
		}
	}
}
