package sem

import (
	"go/parser"
	"go/token"
	"strings"
	"testing"
)

func TestGoRouteLexicalShadowAcrossClosureModes(t *testing.T) {
	for _, prefix := range []string{"package routes\n", ""} {
		for _, mode := range []string{"", "go ", "defer "} {
			for _, closure := range []string{
				`func() { g := 1; _ = &g }()`,
				`func(g int) { _ = &g }(1)`,
				`func() { var g int; child := func() { _ = &g }; _ = child }()`,
				`func() { g := root.Group("/inner"); g = root.Group("/changed") }()`,
			} {
				t.Run(prefix+mode+closure, func(t *testing.T) {
					source := prefix + `func register(root *Router) {
 g := root.Group("/outer")
 g.GET("/before", beforeHandler)
 ` + mode + closure + `
 g.GET("/after", afterHandler)
}`
					assertGoRouteSet(t, goRouteRegistrationsWithin(t, source),
						"/outer/before -> beforeHandler", "/outer/after -> afterHandler")
				})
			}
			t.Run(prefix+mode+"outer escape", func(t *testing.T) {
				source := prefix + `func register(root *Router) {
 g := root.Group("/outer")
 g.GET("/before", beforeHandler)
 ` + mode + `func() { _ = &g }()
 g = root.Group("/rebound")
 g.GET("/after", afterHandler)
}`
				assertGoRouteSet(t, goRouteRegistrationsWithin(t, source), "/outer/before -> beforeHandler")
			})
		}
	}
}

func TestGoRouteLexicalScopeDepthFallbackKeepsBudget(t *testing.T) {
	body := "func register(root *Router) {\n" + strings.Repeat("{\n", 1200) +
		`g := root.Group("/api"); g.GET("/x", handler)` + strings.Repeat("}\n", 1200) + "}\n"
	// Establish that this fixture crosses object resolution's ceiling, but
	// remains valid syntax. The resolver must reach its work budget through
	// the fallback rather than mistake the depth limit for malformed source.
	_, err := parser.ParseFile(token.NewFileSet(), "", "package routes\n"+body, parser.ParseComments)
	if err == nil || !strings.Contains(err.Error(), "scope depth") {
		t.Fatalf("fixture did not reach the object-resolution scope limit: %v", err)
	}
	if _, err := parser.ParseFile(token.NewFileSet(), "", "package routes\n"+body, parser.SkipObjectResolution|parser.ParseComments); err != nil {
		t.Fatal(err)
	}
	for _, prefix := range []string{"package routes\n", ""} {
		got := resolveGoRouteReceiversWithBudget(prefix+body, nil, 1)
		if !got.exhausted || len(got.regs) != 0 {
			t.Fatalf("fallback did not preserve conservative exhaustion: %+v", got)
		}
	}
}
