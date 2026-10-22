package cli

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

// Under a pipe the rows must be the same bytes minus the color: a plain glyph,
// the padded title column, the text, and no escape sequences — that is what
// keeps setup transcripts assertable.
func TestSetupRowsArePlainUnderAPipe(t *testing.T) {
	env := func(key string) string {
		if key == "LC_ALL" {
			return "en_US.UTF-8"
		}
		return ""
	}
	var out bytes.Buffer
	ui := &setupUI{out: &out, color: false, unicode: localeDrawsUnicode(env)}
	ui.Row(setupMarkDone, "agent instructions", "written (AGENTS.md)")
	ui.Row(setupMarkSkipped, "Claude hook", "skipped (--no-claude-hook)")

	got := out.String()
	if strings.Contains(got, "\x1b") {
		t.Fatalf("piped rows must carry no escape sequences:\n%q", got)
	}
	lines := strings.Split(strings.TrimRight(got, "\n"), "\n")
	if lines[0] != "✓ agent instructions written (AGENTS.md)" {
		t.Errorf("done row = %q", lines[0])
	}
	// The title column is padded so texts align across rows.
	if lines[1] != "− Claude hook        skipped (--no-claude-hook)" {
		t.Errorf("skipped row = %q", lines[1])
	}
}

// A locale that cannot draw Unicode gets ASCII glyphs, not replacement boxes.
func TestSetupGlyphsFallBackToASCII(t *testing.T) {
	env := func(key string) string {
		if key == "LC_ALL" {
			return "C"
		}
		return ""
	}
	ui := &setupUI{color: false, unicode: localeDrawsUnicode(env)}
	if g := ui.glyph(setupMarkDone); g != "+" {
		t.Errorf("done glyph = %q, want +", g)
	}
	if g := ui.glyph(setupMarkSkipped); g != "-" {
		t.Errorf("skipped glyph = %q, want -", g)
	}
}

// The command output end to end: one settled row per phase, rerun reported as
// already current, and the opt-out as an explicitly skipped phase.
func TestSetupPrintsOneRowPerPhase(t *testing.T) {
	t.Parallel()
	repo := gitRepoForTest(t, false)
	run := func(extra ...string) string {
		var out bytes.Buffer
		if err := Run(context.Background(), Options{Stdout: &out, Stderr: &out}, append([]string{"setup", "--repo", repo}, extra...)); err != nil {
			t.Fatalf("%v\n%s", err, out.String())
		}
		return out.String()
	}

	first := run()
	for _, want := range []string{"agent instructions", "written (", "Claude hook", "wrote "} {
		if !strings.Contains(first, want) {
			t.Errorf("first run missing %q:\n%s", want, first)
		}
	}
	rerun := run()
	for _, want := range []string{"already current (", "unchanged "} {
		if !strings.Contains(rerun, want) {
			t.Errorf("rerun missing %q:\n%s", want, rerun)
		}
	}
	skipped := run("--no-claude-hook")
	if !strings.Contains(skipped, "skipped (--no-claude-hook)") {
		t.Errorf("opt-out must settle a skipped row:\n%s", skipped)
	}
}
