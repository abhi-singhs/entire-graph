package sem

import (
	"fmt"
	"testing"
)

func TestGoRouteWrappedMountDoesNotInventBareRoute(t *testing.T) {
	for _, target := range []string{"api.With(auth)", "wrap(api)", "api"} {
		t.Run(target, func(t *testing.T) {
			source := fmt.Sprintf(`package p
import "github.com/go-chi/chi/v5"
func routes() {
 r := chi.NewRouter()
 api := chi.NewRouter()
 api.Get("/users", users)
 r.Mount("/v1", %s)
 r.Get("/health", health)
}`, target)
			regs := goRouteRegistrations(source, nil)
			seen := map[string]bool{}
			for _, reg := range regs {
				seen[reg.Route] = true
			}
			if seen["/users"] || !seen["/health"] || seen["/v1/users"] != (target == "api") {
				t.Fatalf("unsafe mount or lost control: %+v", regs)
			}
		})
	}
}
func TestGoRouteClosureParameterDoesNotInventBareRoute(t *testing.T) {
	source := `package p
import "net/http"
func routes() {
 api := http.NewServeMux()
 setup := func(m *http.ServeMux) { m.HandleFunc("/users", users) }
 setup(api)
 http.Handle("/api/", http.StripPrefix("/api", api))
 http.HandleFunc("/health", health)
}`
	regs := goRouteRegistrations(source, nil)
	health := false
	for _, reg := range regs {
		if reg.Route == "/users" {
			t.Fatalf("invented closure parameter route: %+v", regs)
		}
		health = health || reg.Route == "/health"
	}
	if !health {
		t.Fatalf("lost independent root: %+v", regs)
	}
}
