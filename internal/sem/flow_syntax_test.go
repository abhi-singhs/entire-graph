package sem

import "testing"

// Flow extraction must run only the pattern families a language can express.
// Before routing, every family scanned every body: a Python conditional
// pattern over a Go body produced an edge for code that cannot parse in Go —
// a false DATA_FLOWS edge and wasted scan time. A family is skipped only
// where its shape is a syntax error, so no real edge is lost.
func TestFlowExtractionIsLanguageRouted(t *testing.T) {
	flowKinds := func(block, language string) map[string]bool {
		t.Helper()
		kinds := map[string]bool{}
		for _, flow := range returnFlowCalls(newSymbolBody(block), nil, flowSyntaxFor(language)) {
			kinds[flow.EvidenceKind] = true
		}
		return kinds
	}

	// Python conditional shape: an edge in Python, impossible syntax in Go.
	pythonCond := "def pick():\n    v = primary() if ready else fallback()\n    return v\n"
	if !flowKinds(pythonCond, "Python")["conditional_assigned_return_flow"] {
		t.Fatal("Python conditional assignment lost its flow on Python")
	}
	if kinds := flowKinds(pythonCond, "Go"); kinds["conditional_assigned_return_flow"] {
		t.Fatalf("Python conditional pattern matched a Go body: %v", kinds)
	}

	// C ternary shape: an edge in JavaScript, impossible syntax in Python.
	ternary := "function pick() {\n  return ready ? primary() : fallback();\n}\n"
	if !flowKinds(ternary, "JavaScript")["conditional_return_flow"] {
		t.Fatal("ternary return lost its flow on JavaScript")
	}
	if kinds := flowKinds(ternary, "Python"); kinds["conditional_return_flow"] {
		t.Fatalf("ternary pattern matched a Python body: %v", kinds)
	}

	// `||` in value position is legal Go (boolean calls), so Go keeps it.
	logical := "func pick() bool {\n\treturn primary() || fallback()\n}\n"
	if !flowKinds(logical, "Go")["fallback_return_flow"] {
		t.Fatal("Go `||` return flow was dropped; `||` is legal Go and the flow is real")
	}
	if kinds := flowKinds(logical, "Python"); kinds["fallback_return_flow"] {
		t.Fatalf("`||` pattern matched a Python body: %v", kinds)
	}

	// `or` in value position: Python yes, Go no.
	orFallback := "def pick():\n    return primary() or fallback()\n"
	if !flowKinds(orFallback, "Python")["fallback_return_flow"] {
		t.Fatal("Python `or` return flow was dropped")
	}
	if kinds := flowKinds(orFallback, "Go"); kinds["fallback_return_flow"] {
		t.Fatalf("`or` pattern matched a Go body: %v", kinds)
	}

	// An unknown language keeps every family: routing is an optimization that
	// must never drop a real edge under uncertainty.
	if kinds := flowKinds(pythonCond, "SomeFutureLanguage"); !kinds["conditional_assigned_return_flow"] {
		t.Fatalf("unknown language lost a flow family: %v", kinds)
	}
}
