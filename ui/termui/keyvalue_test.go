package termui

import (
	"testing"

	"github.com/moncho/dry/ui"
)

func TestKeyValue(t *testing.T) {
	kvp := NewKeyValuePar("key", "value", &ui.ColorTheme{})
	if kvp == nil {
		t.Error("KeyValuePar is nil")
	}

	if kvp.key.Text != "key:" {
		t.Errorf("KeyValuePar key is not what expected, got: %s", kvp.key.Text)
	}

	if kvp.value.Text != " value" {
		t.Errorf("KeyValuePar value is not what expected, got: %s", kvp.value.Text)
	}

	if len(kvp.Buffer().CellMap) <= 0 {
		t.Errorf("KeyValuePar cellmap is empty content")

	}
}
