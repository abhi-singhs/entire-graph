package cli

import "testing"

// TestIsExploringShellCommandMasksCommandSubstitution: an operator inside `$(...)` or backticks
// belongs to the substituted command, not to the outer statement. Splitting on it cut the head
// stage short and hid the outer `> file` redirect, so a write counted as exploration.
func TestIsExploringShellCommandMasksCommandSubstitution(t *testing.T) {
	t.Parallel()
	writes := []string{
		`cat $(git log --oneline | head -5) > summary.txt`,
		"cat `ls *.go | head -1` > out.txt",
		`cat $(ls; echo x) > out.txt`,
		`cat $(echo $(ls | head -1) | tail -1) > out.txt`,
		`cat $(grep -l ")" x | head -1) > out.txt`,
	}
	for _, command := range writes {
		if isExploringShellCommand(command) {
			t.Fatalf("%q writes a file; it is not a locate call (heads %q)", command, pipelineHeadStages(command))
		}
	}
	reads := []string{
		`cat $(git ls-files | head -5)`,
		`echo $(date) ; grep -rn foo .`,
		"echo `date` && grep -rn foo .",
		`x=$((1+2)); grep -rn foo .`,
	}
	for _, command := range reads {
		if !isExploringShellCommand(command) {
			t.Fatalf("%q reads the repo; it must still count as exploration (heads %q)", command, pipelineHeadStages(command))
		}
	}
	// An unterminated substitution falls back to the unmasked reading, like an unbalanced quote.
	for index, masked := range shellQuoteMask([]rune(`echo $(unterminated; grep -rn foo .`)) {
		if masked {
			t.Fatalf("an unbalanced substitution masked index %d; the whole mask must fall back to false", index)
		}
	}
}
