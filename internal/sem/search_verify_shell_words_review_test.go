package sem

import "testing"

func TestSearchVerifyScriptPreservesQuotedWordBoundaries(t *testing.T) {
	for _, tc := range []struct {
		script string
		want   bool
	}{
		{`MSG="hello jest world" echo done`, false},
		{`MSG="hello jest world" jest`, true},
		{`MSG='hello jest world' echo done`, false},
		{`MSG=hello\ jest\ world echo done`, false},
		{`cross-env MSG="hello jest world" jest`, true},
		{`bunx jest`, true}, {`bun run jest`, true}, {`env run jest`, false},
		{`echo "jest"`, false}, {`"jest"`, true},
	} {
		if got := searchVerifyScriptInvokesRunner(tc.script, "jest"); got != tc.want {
			t.Errorf("%s: got %v want %v", tc.script, got, tc.want)
		}
	}
}
