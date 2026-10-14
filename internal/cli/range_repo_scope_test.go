package cli

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRangeCommandsRejectSubdirectoryRepo(t *testing.T) {
	repo := t.TempDir()
	git(t, repo, "init")
	subdir := filepath.Join(repo, "sub")
	if err := os.Mkdir(subdir, 0o755); err != nil {
		t.Fatal(err)
	}

	if err := requireRangeRepoRoot(t.Context(), repo, "diff"); err != nil {
		t.Fatalf("checkout root rejected: %v", err)
	}
	for _, command := range []string{"diff", "analyze", "commit", "checkpoint"} {
		t.Run(command, func(t *testing.T) {
			args := []string{command, "--repo", subdir}
			if command == "checkpoint" {
				args = append(args, "checkpoint-id")
			}
			err := Run(t.Context(), Options{Stdout: io.Discard, Stderr: io.Discard}, args)
			if err == nil || !strings.Contains(err.Error(), "checkout root") {
				t.Fatalf("%s with a subdirectory returned %v, want checkout-root refusal", command, err)
			}
		})
	}
}
