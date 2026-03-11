package parser

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/goatlas/goatlas/internal/indexer/domain"
)

const ginFixture = `package routes

import "github.com/gin-gonic/gin"

func SetupRouter(r *gin.Engine) {
	r.GET("/users", ListUsers)
	r.POST("/users", CreateUser)
	r.PUT("/users/:id", UpdateUser)
	r.DELETE("/users/:id", DeleteUser)
}
`

const netHttpFixture = `package server

import "net/http"

func RegisterHandlers(mux *http.ServeMux) {
	mux.HandleFunc("/api/ping", PingHandler)
	mux.HandleFunc("/api/users", UsersHandler)
}
`

const noFrameworkFixture = `package plain

import "fmt"

func DoSomething() {
	fmt.Println("no routes here")
}
`

func writeFixture(t *testing.T, name, content string) string {
	t.Helper()
	dir := t.TempDir()
	f := filepath.Join(dir, name)
	if err := os.WriteFile(f, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	return f
}

func TestExtractRoutes_Gin(t *testing.T) {
	f := writeFixture(t, "routes.go", ginFixture)
	imports := []domain.Import{{ImportPath: "github.com/gin-gonic/gin"}}

	endpoints, err := ExtractRoutes(f, imports)
	if err != nil {
		t.Fatalf("ExtractRoutes error: %v", err)
	}

	if len(endpoints) == 0 {
		t.Fatal("expected at least one endpoint")
	}

	methods := map[string]bool{}
	paths := map[string]bool{}
	for _, ep := range endpoints {
		methods[ep.Method] = true
		paths[ep.Path] = true
		if ep.Framework != "gin" {
			t.Errorf("expected framework 'gin', got %q", ep.Framework)
		}
	}

	for _, m := range []string{"GET", "POST", "PUT", "DELETE"} {
		if !methods[m] {
			t.Errorf("expected method %q", m)
		}
	}
	if !paths["/users"] {
		t.Error("expected path '/users'")
	}
}

func TestExtractRoutes_NetHttp(t *testing.T) {
	f := writeFixture(t, "server.go", netHttpFixture)
	imports := []domain.Import{{ImportPath: "net/http"}}

	endpoints, err := ExtractRoutes(f, imports)
	if err != nil {
		t.Fatalf("ExtractRoutes error: %v", err)
	}

	if len(endpoints) < 2 {
		t.Fatalf("expected >=2 endpoints, got %d", len(endpoints))
	}

	for _, ep := range endpoints {
		if ep.Framework != "net_http" {
			t.Errorf("expected framework 'net_http', got %q", ep.Framework)
		}
	}
}

func TestExtractRoutes_NoFramework(t *testing.T) {
	f := writeFixture(t, "plain.go", noFrameworkFixture)
	imports := []domain.Import{{ImportPath: "fmt"}}

	endpoints, err := ExtractRoutes(f, imports)
	if err != nil {
		t.Fatalf("ExtractRoutes error: %v", err)
	}
	if len(endpoints) != 0 {
		t.Errorf("expected 0 endpoints for plain package, got %d", len(endpoints))
	}
}

func TestDetectFramework(t *testing.T) {
	cases := []struct {
		imports  []domain.Import
		expected string
	}{
		{[]domain.Import{{ImportPath: "github.com/gin-gonic/gin"}}, "gin"},
		{[]domain.Import{{ImportPath: "github.com/labstack/echo/v4"}}, "echo"},
		{[]domain.Import{{ImportPath: "github.com/go-chi/chi/v5"}}, "chi"},
		{[]domain.Import{{ImportPath: "net/http"}}, "net_http"},
		{[]domain.Import{{ImportPath: "fmt"}}, ""},
	}

	for _, tc := range cases {
		got := detectFramework(tc.imports)
		if got != tc.expected {
			t.Errorf("detectFramework(%v) = %q, want %q", tc.imports, got, tc.expected)
		}
	}
}
