package termui

import "testing"

const text = "Move along, nothing to see here"

func TestParColumn(t *testing.T) {
	p := NewParColumn(text)

	if p == nil {
		t.Error("ParColumn is nil")
	}

	if p.Text != text {
		t.Errorf("ParColumn content is not what was expected: %s", p.Text)
	}
	if p.Border {
		t.Error("ParColumn has a border")

	}
}
