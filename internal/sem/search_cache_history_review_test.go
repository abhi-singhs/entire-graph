package sem

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestSearchCacheRefreshesHistoryAtTheSameTree(t *testing.T) {
	repo := t.TempDir()
	sharedCacheFixture(t, repo)
	cache := t.TempDir()
	options := ProviderSnapshotOptions{Profile: ProfileFull}
	warm, hit, err := PreindexProviderSnapshot(t.Context(), repo, "test-version", options, cache)
	if err != nil || hit {
		t.Fatalf("warm: %v %v", hit, err)
	}
	paths := []string{"ledger/ledger.go", "web/client.ts"}
	originals := make(map[string][]byte)
	for _, path := range paths {
		content, err := os.ReadFile(filepath.Join(repo, path))
		if err != nil {
			t.Fatal(err)
		}
		originals[path] = content
		write(t, repo, path, string(content)+"\n// history change\n")
	}
	git(t, repo, "add", ".")
	git(t, repo, "commit", "-q", "-m", "history change")
	for _, path := range paths {
		write(t, repo, path, string(originals[path]))
	}
	git(t, repo, "add", ".")
	git(t, repo, "commit", "-q", "-m", "restore tree")
	served, hit, err := PreindexProviderSnapshot(t.Context(), repo, "test-version", options, cache)
	if err != nil || !hit {
		t.Fatalf("same-tree load: %v %v", hit, err)
	}
	cold, err := BuildProviderSnapshotWithOptions(t.Context(), repo, "test-version", options)
	if err != nil {
		t.Fatal(err)
	}
	history := func(s ProviderSnapshot) []RelationRecord {
		var out []RelationRecord
		for _, r := range s.Relations {
			if r.Type == "FILE_CHANGES_WITH" {
				out = append(out, r)
			}
		}
		return out
	}
	if reflect.DeepEqual(history(warm), history(cold)) {
		t.Fatal("fixture histories did not differ")
	}
	if !reflect.DeepEqual(history(served), history(cold)) || served.Header.Stats.Relations != cold.Header.Stats.Relations || !reflect.DeepEqual(served.Header.Completeness.Relations, cold.Header.Completeness.Relations) {
		t.Fatal("cache reused another history or stale counts")
	}
}
