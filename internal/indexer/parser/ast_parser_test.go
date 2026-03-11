package parser

import (
	"os"
	"path/filepath"
	"testing"
)

const fixtureGo = `package mypackage

import (
	"fmt"
	net "net/http"
)

// MyStruct is a test struct.
type MyStruct struct {
	Name string
}

// MyInterface is a test interface.
type MyInterface interface {
	DoSomething() error
}

const MaxRetries = 3

var DefaultTimeout = 30

// Hello returns a greeting.
func Hello(name string) string {
	return fmt.Sprintf("Hello, %s", name)
}

// (m *MyStruct) Greet is a method.
func (m *MyStruct) Greet() string {
	return "Hi from " + m.Name
}

func multiReturn(a, b int) (int, error) {
	_ = net.StatusOK
	return a + b, nil
}
`

func TestParseFile_Symbols(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "fixture.go")
	if err := os.WriteFile(f, []byte(fixtureGo), 0600); err != nil {
		t.Fatal(err)
	}

	result, err := ParseFile(f)
	if err != nil {
		t.Fatalf("ParseFile error: %v", err)
	}

	if result.Module != "mypackage" {
		t.Errorf("expected module 'mypackage', got %q", result.Module)
	}

	kinds := map[string]int{}
	names := map[string]bool{}
	for _, s := range result.Symbols {
		kinds[s.Kind]++
		names[s.Name] = true
	}

	if !names["Hello"] {
		t.Error("expected symbol 'Hello'")
	}
	if !names["Greet"] {
		t.Error("expected symbol 'Greet'")
	}
	if !names["multiReturn"] {
		t.Error("expected symbol 'multiReturn'")
	}
	if !names["MyStruct"] {
		t.Error("expected symbol 'MyStruct'")
	}
	if !names["MyInterface"] {
		t.Error("expected symbol 'MyInterface'")
	}
	if !names["MaxRetries"] {
		t.Error("expected symbol 'MaxRetries'")
	}
	if !names["DefaultTimeout"] {
		t.Error("expected symbol 'DefaultTimeout'")
	}

	if kinds["func"] < 1 {
		t.Error("expected at least 1 func")
	}
	if kinds["method"] < 1 {
		t.Error("expected at least 1 method")
	}
	if kinds["struct"] < 1 {
		t.Error("expected at least 1 struct")
	}
	if kinds["interface"] < 1 {
		t.Error("expected at least 1 interface")
	}
}

func TestParseFile_Imports(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "fixture.go")
	if err := os.WriteFile(f, []byte(fixtureGo), 0600); err != nil {
		t.Fatal(err)
	}

	result, err := ParseFile(f)
	if err != nil {
		t.Fatalf("ParseFile error: %v", err)
	}

	paths := map[string]bool{}
	aliases := map[string]bool{}
	for _, imp := range result.Imports {
		paths[imp.ImportPath] = true
		if imp.Alias != "" {
			aliases[imp.Alias] = true
		}
	}

	if !paths["fmt"] {
		t.Error("expected import 'fmt'")
	}
	if !paths["net/http"] {
		t.Error("expected import 'net/http'")
	}
	if !aliases["net"] {
		t.Error("expected alias 'net' for net/http")
	}
}

func TestParseFile_MethodReceiver(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "fixture.go")
	if err := os.WriteFile(f, []byte(fixtureGo), 0600); err != nil {
		t.Fatal(err)
	}

	result, err := ParseFile(f)
	if err != nil {
		t.Fatalf("ParseFile error: %v", err)
	}

	for _, s := range result.Symbols {
		if s.Name == "Greet" {
			if s.Kind != "method" {
				t.Errorf("Greet should be method, got %q", s.Kind)
			}
			if s.Receiver == "" {
				t.Error("Greet should have a receiver")
			}
			return
		}
	}
	t.Error("symbol 'Greet' not found")
}

func TestParseFile_DocComment(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "fixture.go")
	if err := os.WriteFile(f, []byte(fixtureGo), 0600); err != nil {
		t.Fatal(err)
	}

	result, err := ParseFile(f)
	if err != nil {
		t.Fatalf("ParseFile error: %v", err)
	}

	for _, s := range result.Symbols {
		if s.Name == "Hello" {
			if s.DocComment == "" {
				t.Error("Hello should have doc comment")
			}
			return
		}
	}
	t.Error("symbol 'Hello' not found")
}

func TestParseFile_InvalidFile(t *testing.T) {
	_, err := ParseFile("/nonexistent/file.go")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestModuleFromGoMod(t *testing.T) {
	dir := t.TempDir()
	content := "module github.com/example/myapp\n\ngo 1.21\n"
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(content), 0600); err != nil {
		t.Fatal(err)
	}

	mod := ModuleFromGoMod(dir)
	if mod != "github.com/example/myapp" {
		t.Errorf("expected 'github.com/example/myapp', got %q", mod)
	}
}

func TestModuleFromGoMod_Missing(t *testing.T) {
	mod := ModuleFromGoMod("/nonexistent/dir")
	if mod != "" {
		t.Errorf("expected empty string, got %q", mod)
	}
}
