package sem

import "testing"

func TestJuliaHeaderSignatureIdentityIsRevisioned(t *testing.T) {
	parse := func(source string) []SymbolRecord {
		t.Helper()
		entities, _, status := (TreeSitterParser{}).ParseWithStatus("overloads.jl", source)
		if status.ParseError || len(entities) != 2 {
			t.Fatalf("parse = %#v, status = %#v", entities, status)
		}
		return entitySymbols("local/example", "overloads.jl", "Julia", entities)
	}
	before := parse("f(x::Int) = x + 1\nf(x::String) = length(x)\n")
	after := parse("f(x::String) = length(x)\nf(x::Int) = x + 2\n")
	ids := map[string]string{}
	for _, symbol := range before {
		want := "local/example:Julia:overloads.jl:function:f#sig:" + hash(symbol.Signature)
		if symbol.ID != want {
			t.Fatalf("header identity = %q, want %q", symbol.ID, want)
		}
		ids[symbol.Signature] = symbol.ID
	}
	for _, symbol := range after {
		if symbol.ID != ids[symbol.Signature] {
			t.Errorf("body edit/reordering changed identity: %q -> %q", ids[symbol.Signature], symbol.ID)
		}
	}
	legacyID := "local/example:Julia:overloads.jl:function:f#sig:" + hash("f(x::Int) = x + 1")
	if before[0].ID == legacyID {
		t.Fatal("Julia overload still uses the body-laden signature for identity")
	}
	if IdentityRevision == "4" {
		t.Fatal("Julia header signatures re-key overloads and must invalidate identity revision 4")
	}
}

func TestJuliaOneLineRenameRetainsBodyEvidence(t *testing.T) {
	for _, form := range []struct {
		name, before, renamed, replaced string
	}{
		{"short", "encode(x) = x + 1", "send(x) = x + 1", "send(x) = error(\"unavailable\")"},
		{"long", "function encode(x); x + 1; end", "function send(x); x + 1; end", "function send(x); error(\"unavailable\"); end"},
	} {
		t.Run(form.name, func(t *testing.T) {
			parse := func(source string) []Entity {
				t.Helper()
				entities, _, status := (TreeSitterParser{}).ParseWithStatus("rename.jl", source+"\n")
				if status.ParseError || len(entities) != 1 {
					t.Fatalf("parse = %#v, status = %#v", entities, status)
				}
				return entities
			}
			before := parse(form.before)
			changes, removed, added := compareEntities(before, parse(form.replaced), false)
			if len(changes) != 0 || len(removed) != 1 || len(added) != 1 {
				t.Fatalf("unrelated replacement: changes=%#v, removed=%#v, added=%#v", changes, removed, added)
			}
			changes, removed, added = compareEntities(before, parse(form.renamed), false)
			if len(changes) != 1 || changes[0].Type != "renamed" || len(removed) != 0 || len(added) != 0 {
				t.Fatalf("genuine rename: changes=%#v, removed=%#v, added=%#v", changes, removed, added)
			}
		})
	}
}
