package upgrade

import (
	"bytes"
	"strings"
	"testing"
)

func TestConfirm_EnvAssumeYes(t *testing.T) {
	t.Setenv("KDEPS_YES", "1")
	if !Confirm(&bytes.Buffer{}, strings.NewReader(""), false, "proceed? ") {
		t.Fatal("KDEPS_YES=1 must always confirm, even non-interactive")
	}
}

func TestConfirm_EnvAssumeYesAlt(t *testing.T) {
	t.Setenv("KDEPS_ASSUME_YES", "1")
	if !Confirm(&bytes.Buffer{}, strings.NewReader(""), false, "proceed? ") {
		t.Fatal("KDEPS_ASSUME_YES=1 must always confirm")
	}
}

func TestConfirm_NonInteractive_NoEnv(t *testing.T) {
	if Confirm(&bytes.Buffer{}, strings.NewReader("y\n"), false, "proceed? ") {
		t.Fatal("non-interactive without the env override must never confirm, regardless of stdin")
	}
}

func TestConfirm_Interactive_Responses(t *testing.T) {
	cases := map[string]bool{
		"y\n":    true,
		"Y\n":    true,
		"yes\n":  true,
		"YES\n":  true,
		"\n":     true, // blank line defaults to yes
		"n\n":    false,
		"no\n":   false,
		"nope\n": false,
	}
	for input, want := range cases {
		var out bytes.Buffer
		got := Confirm(&out, strings.NewReader(input), true, "proceed? ")
		if got != want {
			t.Errorf("Confirm with input %q = %v, want %v", input, got, want)
		}
		if !strings.Contains(out.String(), "proceed? ") {
			t.Errorf("prompt must be written to w for input %q", input)
		}
	}
}

func TestConfirm_Interactive_EOFWithNoInput(t *testing.T) {
	// An empty reader hits EOF immediately with no bytes read.
	if Confirm(&bytes.Buffer{}, strings.NewReader(""), true, "proceed? ") {
		t.Fatal("EOF with no input must not confirm")
	}
}
