package sem

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// The store's whole contract is fidelity: substituting it for the gzipped JSON
// entry must be invisible. The pin round-trips one real snapshot through BOTH
// containers and requires the results to be deeply equal — covering every
// record stream, every side map the private symbol fields travel in, and the
// envelope scalars.
func TestSearchStoreRoundTripsSnapshot(t *testing.T) {
	dir := t.TempDir()
	write := func(name, content string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	// A corpus small enough to build in milliseconds but wide enough to
	// populate every table: nested (Local) and overload-style symbols, calls,
	// types, aliases via a tool registration, and a second language.
	write("main.go", "package sample\n\ntype Widget struct{ Name string }\n\nfunc NewWidget(name string) *Widget { return &Widget{Name: name} }\n\nfunc Render(w *Widget) string {\n\thelper := func() string { return w.Name }\n\treturn helper()\n}\n")
	write("util.py", "def fetch(url):\n    return url\n\ndef main():\n    v = fetch(\"x\")\n    return v\n")
	write("lib.ts", "export function load(path: string): string;\nexport function load(path: string, mode: string): string;\nexport function load(path: string, mode?: string): string { return path; }\n")

	snapshot, err := BuildProviderSnapshotWithOptions(t.Context(), dir, "store-test", ProviderSnapshotOptions{Worktree: true})
	if err != nil {
		t.Fatal(err)
	}
	cache := newCachedSearchSnapshot("store-test", "commit-a", "tree-a", ProviderSnapshotOptions{}, snapshot, t.Context())

	// Reference: the gz JSON container production serves today.
	entry, err := newCacheEntry(t.TempDir(), "search", "test-v1", validTestCacheKey(t))
	if err != nil {
		t.Fatal(err)
	}
	if err := writeSearchSnapshot(entry, cache); err != nil {
		t.Fatal(err)
	}
	fromJSON, err := readSearchSnapshot(entry)
	if err != nil {
		t.Fatal(err)
	}

	storePath := filepath.Join(t.TempDir(), "store.sqlite")
	if err := buildSearchStore(t.Context(), storePath, "digest-a", &cache); err != nil {
		t.Fatal(err)
	}
	if _, err := loadSearchStore(t.Context(), storePath, "digest-b"); err == nil {
		t.Fatal("store served a snapshot for an entry digest it was not built against")
	}
	fromStore, err := loadSearchStore(t.Context(), storePath, "digest-a")
	if err != nil {
		t.Fatal(err)
	}

	// Direct DeepEqual, no normalization: the loader must reproduce the exact
	// nil-vs-empty shapes and slice orders the JSON path produces, or a real
	// fidelity regression could hide behind a test-side normalizer.
	if !reflect.DeepEqual(fromStore, fromJSON) {
		t.Fatal(diffSearchStoreCaches(t, fromStore, fromJSON))
	}
}

func diffSearchStoreCaches(t *testing.T, g, w cachedSearchSnapshot) string {
	t.Helper()
	report := "store round-trip differs from JSON round-trip:"
	check := func(name string, gv, wv any) {
		if !reflect.DeepEqual(gv, wv) {
			report += "\n  " + name
		}
	}
	check("envelope scalars", [8]any{g.CacheVersion, g.ProviderVersion, g.Commit, g.Tree, g.Profile, g.MaxParseBytes, g.DerivedFrom, g.RepoScope},
		[8]any{w.CacheVersion, w.ProviderVersion, w.Commit, w.Tree, w.Profile, w.MaxParseBytes, w.DerivedFrom, w.RepoScope})
	check("header", g.Snapshot.Header, w.Snapshot.Header)
	check("files", g.Snapshot.Files, w.Snapshot.Files)
	check("externals", g.Snapshot.Externals, w.Snapshot.Externals)
	check("symbols", g.Snapshot.Symbols, w.Snapshot.Symbols)
	check("relations", g.Snapshot.Relations, w.Snapshot.Relations)
	check("file lines", g.FileLines, w.FileLines)
	check("byte ranges", g.SymbolByteRanges, w.SymbolByteRanges)
	check("name lines", g.SymbolNameLines, w.SymbolNameLines)
	check("parameter names", g.SymbolParameterNames, w.SymbolParameterNames)
	check("parameter known ids", g.SymbolParameterNamesKnownIDs, w.SymbolParameterNamesKnownIDs)
	check("signature types", g.SymbolSignatureTypes, w.SymbolSignatureTypes)
	check("local ids", g.LocalSymbolIDs, w.LocalSymbolIDs)
	check("bodyless ids", g.BodylessSymbolIDs, w.BodylessSymbolIDs)
	check("unmerged edges", g.UnmergedDataFlowEdges, w.UnmergedDataFlowEdges)
	return report
}

func validTestCacheKey(t *testing.T) string {
	t.Helper()
	return "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
}
