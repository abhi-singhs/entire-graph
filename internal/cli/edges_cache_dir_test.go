package cli

import (
	"bytes"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// recordsCacheEntries lists the provider-records entries under a cache directory.
func recordsCacheEntries(t *testing.T, cacheDir string) []string {
	t.Helper()
	var entries []string
	root := filepath.Join(cacheDir, "records")
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() && strings.HasSuffix(path, ".json.gz") {
			entries = append(entries, path)
		}
		return nil
	})
	if err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	return entries
}

// TestTargetedEdgesHonourTheCacheDir pins `--cache-dir` on a targeted edge query. It used to be
// parsed and then ignored: the --to/--from/--relation path took no cache at all, so every "callers
// of X" rebuilt the whole graph and the flag said nothing about where (or whether) anything went.
//
// It now reads and fills the same records entry an unfiltered `edges` run does. Pinned: the entry
// lands in the named directory and nowhere else, the second query is served from it (the entry is
// the same file, not a rewrite), and a hit is byte-identical to the uncached answer.
func TestTargetedEdgesHonourTheCacheDir(t *testing.T) {
	repo := t.TempDir()
	git(t, repo, "init")
	git(t, repo, "config", "user.name", "Entire Graph Test")
	git(t, repo, "config", "user.email", "graph@example.com")
	write(t, repo, "main.go", `package sample

func helper() int { return 1 }

func Main() int { return helper() }

func Other() int { return helper() + Main() }
`)
	git(t, repo, "add", "main.go")
	git(t, repo, "commit", "-m", "initial")
	cacheDir, pluginDir := t.TempDir(), t.TempDir()

	run := func(args ...string) (string, string) {
		t.Helper()
		var out, errOut bytes.Buffer
		if err := Run(t.Context(), Options{
			Version: "edges-cache-dir-test",
			Env:     EntireEnv{RepoRoot: repo, PluginDataDir: pluginDir},
			Stdout:  &out,
			Stderr:  &errOut,
		}, append([]string{"edges", "--repo", repo, "--format", "ndjson"}, args...)); err != nil {
			t.Fatalf("edges %v: %v\n%s", args, err, errOut.String())
		}
		return out.String(), errOut.String()
	}
	targeted := []string{"--to", "helper", "--relation", "CALLS"}

	uncached, uncachedErr := run(append(targeted, "--no-cache")...)
	if strings.Count(uncached, `"record_type":"relation"`) != 2 {
		t.Fatalf("fixture should have two CALLS edges into helper:\n%s", uncached)
	}
	if entries := recordsCacheEntries(t, cacheDir); len(entries) != 0 {
		t.Fatalf("--no-cache wrote %v", entries)
	}

	cold, coldErr := run(append(targeted, "--cache-dir", cacheDir)...)
	entries := recordsCacheEntries(t, cacheDir)
	if len(entries) != 1 {
		t.Fatalf("targeted edges with --cache-dir wrote %d entries under it, want 1", len(entries))
	}
	if other := recordsCacheEntries(t, pluginDir); len(other) != 0 {
		t.Fatalf("targeted edges wrote to the default cache %v despite --cache-dir", other)
	}
	before, err := os.Stat(entries[0])
	if err != nil {
		t.Fatal(err)
	}

	warm, warmErr := run(append(targeted, "--cache-dir", cacheDir)...)
	after, err := os.Stat(entries[0])
	if err != nil {
		t.Fatal(err)
	}
	if !os.SameFile(before, after) || !after.ModTime().Equal(before.ModTime()) {
		t.Fatal("the second targeted query rewrote the entry instead of being served from it")
	}
	for _, got := range []struct{ name, out, errOut string }{
		{"cold", cold, coldErr},
		{"warm", warm, warmErr},
	} {
		if got.out != uncached {
			t.Fatalf("%s output differs from the uncached answer:\n--- uncached\n%s\n--- %s\n%s", got.name, uncached, got.name, got.out)
		}
		if got.errOut != uncachedErr {
			t.Fatalf("%s stderr = %q, want %q", got.name, got.errOut, uncachedErr)
		}
	}

	// The entry is the unfiltered `edges` stream, so an unfiltered run is a hit on it too.
	full, _ := run("--cache-dir", cacheDir)
	if again := recordsCacheEntries(t, cacheDir); len(again) != 1 {
		t.Fatalf("unfiltered edges wrote a second entry %v instead of reusing the targeted query's", again)
	}
	fullUncached, _ := run("--no-cache")
	if full != fullUncached {
		t.Fatalf("unfiltered edges served from the targeted query's entry differs from uncached:\n--- uncached\n%s\n--- cached\n%s", fullUncached, full)
	}
}

// An entry the filter cannot read is not trusted: the caller rebuilds instead of printing a
// partial answer from it. Blank lines carry nothing and are skipped.
func TestFilterCachedEdgeRecordsRefusesAnUnreadableEntry(t *testing.T) {
	t.Parallel()
	flags := providerFlags{To: "helper"}
	header := `{"schema_version":"1.3"}` + "\n"
	match := `{"record_type":"relation","from_id":"r:Go:a.go:function:Main","to_id":"r:Go:a.go:function:helper","type":"CALLS"}` + "\n"
	miss := `{"record_type":"relation","from_id":"r:Go:a.go:function:Main","to_id":"r:Go:a.go:function:Other","type":"CALLS"}` + "\n"
	got, matched, ok := filterCachedEdgeRecords([]byte(header+"\n"+match+miss), flags)
	if !ok || matched != 1 || string(got) != header+match {
		t.Fatalf("filtered = %q matched=%d ok=%v", got, matched, ok)
	}
	for _, broken := range []string{header + "{not json\n", header + `{"record_type":"relation","to_id":7}` + "\n"} {
		if _, _, ok := filterCachedEdgeRecords([]byte(broken), flags); ok {
			t.Fatalf("an unreadable entry was accepted: %q", broken)
		}
	}
}
