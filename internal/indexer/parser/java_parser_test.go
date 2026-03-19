package parser

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseJavaFile_symbols(t *testing.T) {
	dir := t.TempDir()
	src := `package com.example.svc;

public class UserService {
    public String getUser(String id) {
        return id;
    }

    public void createUser(String name) {}
}
`
	f := filepath.Join(dir, "UserService.java")
	os.WriteFile(f, []byte(src), 0644)

	result, err := ParseJavaFile(f)
	if err != nil {
		t.Fatal(err)
	}

	if len(result.Symbols) == 0 {
		t.Fatal("expected symbols")
	}

	hasClass := false
	hasMethods := 0
	for _, s := range result.Symbols {
		if s.Kind == "class" && s.Name == "UserService" {
			hasClass = true
		}
		if s.Kind == "method" {
			hasMethods++
		}
	}
	if !hasClass {
		t.Error("expected UserService class symbol")
	}
	if hasMethods < 2 {
		t.Errorf("expected ≥2 methods, got %d", hasMethods)
	}
}

func TestParseJavaFile_imports(t *testing.T) {
	dir := t.TempDir()
	src := `package com.example;

import io.grpc.ManagedChannel;
import org.springframework.kafka.annotation.KafkaListener;

public class App {}
`
	f := filepath.Join(dir, "App.java")
	os.WriteFile(f, []byte(src), 0644)

	result, err := ParseJavaFile(f)
	if err != nil {
		t.Fatal(err)
	}

	if len(result.Imports) < 2 {
		t.Fatalf("expected ≥2 imports, got %d", len(result.Imports))
	}
}

func TestParseJavaFile_springEndpoints(t *testing.T) {
	dir := t.TempDir()
	src := `package com.example;

import org.springframework.web.bind.annotation.*;

@RestController
@RequestMapping("/users")
public class UserController {
    @GetMapping("/{id}")
    public String getUser(String id) {
        return id;
    }
}
`
	f := filepath.Join(dir, "UserController.java")
	os.WriteFile(f, []byte(src), 0644)

	result, err := ParseJavaFile(f)
	if err != nil {
		t.Fatal(err)
	}

	if len(result.Endpoints) == 0 {
		t.Error("expected Spring MVC endpoints from @GetMapping")
	}
}
