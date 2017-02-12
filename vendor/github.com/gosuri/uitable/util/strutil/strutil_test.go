package strutil

import (
	"testing"
)

func TestResize(t *testing.T) {
	s := "foo"
	got := Resize(s, 5)
	if len(got) != 5 {
		t.Fatal("want", 5, "got", len(got))
	}
	s = "foobar"
	got = Resize(s, 5)

	if got != "fo..." {
		t.Fatal("want", "fo...", "got", got)
	}
}

func TestJoin(t *testing.T) {
	got := Join([]string{"foo", "bar"}, ",")
	if got != "foo,bar" {
		t.Fatal("want", "foo,bar", "got", got)
	}
}

func TestPadRight(t *testing.T) {
	got := PadRight("foo", 5, '-')
	if got != "foo--" {
		t.Fatal("want", "foo--", "got", got)
	}
}

func TestPadLeft(t *testing.T) {
	got := PadLeft("foo", 5, '-')
	if got != "--foo" {
		t.Fatal("want", "--foo", "got", got)
	}
}
