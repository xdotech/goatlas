package parser

import (
	"os"
	"path/filepath"
	"testing"
)

const pythonFixture = `"""Module docstring for the whole file."""

import os
import os.path
import json as j
from pathlib import Path
from collections import OrderedDict
from . import utils
from ..models import User

# Constants
MAX_RETRIES = 3
DEFAULT_TIMEOUT = 30
API_BASE_URL = "https://api.example.com"

# Module-level variable
default_config = {}
__version__ = "1.0.0"


class BaseService:
    """Base service class."""
    pass


class UserService(BaseService):
    """Service for managing users.

    Handles CRUD operations and authentication.
    """

    MAX_USERS = 100

    def __init__(self, db):
        """Initialize the service."""
        self.db = db

    def get_user(self, user_id: int) -> dict:
        """Retrieve a user by ID."""
        return self.db.find(user_id)

    def create_user(self, name: str, email: str) -> dict:
        return {"name": name, "email": email}

    @staticmethod
    def validate_email(email: str) -> bool:
        """Check if email is valid."""
        return "@" in email

    @classmethod
    def from_config(cls, config: dict) -> "UserService":
        return cls(config["db"])


def process_order(order_id: int, items: list) -> bool:
    """Process an order with the given items."""
    return True


def helper_func():
    pass


def calculate_total(prices: list[float], tax_rate: float = 0.1) -> float:
    """Calculate total with tax."""
    subtotal = sum(prices)
    return subtotal * (1 + tax_rate)


class NestedExample:
    class InnerConfig:
        """Inner configuration class."""
        DEBUG = False
`

const pythonFlaskFixture = `from flask import Flask, jsonify

app = Flask(__name__)

@app.route("/users", methods=["GET"])
def list_users():
    """List all users."""
    return jsonify([])

@app.route("/users/<int:user_id>", methods=["GET", "POST"])
def get_or_create_user(user_id):
    return jsonify({})

@app.route("/health")
def health_check():
    return "ok"
`

const pythonFastAPIFixture = `from fastapi import FastAPI, APIRouter

app = FastAPI()
router = APIRouter()

@router.get("/items/{item_id}")
def get_item(item_id: int):
    """Get a single item."""
    return {"item_id": item_id}

@router.post("/items")
def create_item(name: str):
    return {"name": name}

@app.delete("/items/{item_id}")
def delete_item(item_id: int):
    return {"deleted": True}
`

func writePythonFixture(t *testing.T, name, content string) string {
	t.Helper()
	dir := t.TempDir()
	f := filepath.Join(dir, name)
	if err := os.WriteFile(f, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	return f
}

func TestParsePythonFile_Classes(t *testing.T) {
	f := writePythonFixture(t, "service.py", pythonFixture)
	result, err := ParsePythonFile(f)
	if err != nil {
		t.Fatalf("ParsePythonFile error: %v", err)
	}

	names := map[string]string{}
	for _, s := range result.Symbols {
		names[s.Name] = s.Kind
	}

	classes := []string{"BaseService", "UserService", "NestedExample", "InnerConfig"}
	for _, c := range classes {
		if kind, ok := names[c]; !ok {
			t.Errorf("expected class %q not found", c)
		} else if kind != KindClass {
			t.Errorf("%q: expected kind=%q, got %q", c, KindClass, kind)
		}
	}
}

func TestParsePythonFile_Functions(t *testing.T) {
	f := writePythonFixture(t, "service.py", pythonFixture)
	result, err := ParsePythonFile(f)
	if err != nil {
		t.Fatalf("ParsePythonFile error: %v", err)
	}

	names := map[string]string{}
	for _, s := range result.Symbols {
		names[s.Name] = s.Kind
	}

	// Module-level functions
	funcs := []string{"process_order", "helper_func", "calculate_total"}
	for _, fn := range funcs {
		if kind, ok := names[fn]; !ok {
			t.Errorf("expected function %q not found", fn)
		} else if kind != KindFunc {
			t.Errorf("%q: expected kind=%q, got %q", fn, KindFunc, kind)
		}
	}
}

func TestParsePythonFile_Methods(t *testing.T) {
	f := writePythonFixture(t, "service.py", pythonFixture)
	result, err := ParsePythonFile(f)
	if err != nil {
		t.Fatalf("ParsePythonFile error: %v", err)
	}

	names := map[string]string{}
	receivers := map[string]string{}
	for _, s := range result.Symbols {
		names[s.Name] = s.Kind
		if s.Receiver != "" {
			receivers[s.Name] = s.Receiver
		}
	}

	methods := []string{"__init__", "get_user", "create_user", "validate_email", "from_config"}
	for _, m := range methods {
		if kind, ok := names[m]; !ok {
			t.Errorf("expected method %q not found", m)
		} else if kind != KindMethod {
			t.Errorf("%q: expected kind=%q, got %q", m, KindMethod, kind)
		}
	}

	// Check receivers
	if recv := receivers["get_user"]; recv != "UserService" {
		t.Errorf("get_user: expected receiver 'UserService', got %q", recv)
	}
}

func TestParsePythonFile_Imports(t *testing.T) {
	f := writePythonFixture(t, "service.py", pythonFixture)
	result, err := ParsePythonFile(f)
	if err != nil {
		t.Fatalf("ParsePythonFile error: %v", err)
	}

	paths := map[string]bool{}
	aliases := map[string]bool{}
	for _, imp := range result.Imports {
		paths[imp.ImportPath] = true
		if imp.Alias != "" {
			aliases[imp.Alias] = true
		}
	}

	expectedPaths := []string{"os", "os.path", "pathlib", "collections"}
	for _, p := range expectedPaths {
		if !paths[p] {
			t.Errorf("expected import %q not found (have: %v)", p, paths)
		}
	}

	if !aliases["j"] {
		t.Error("expected alias 'j' for json")
	}
}

func TestParsePythonFile_Constants(t *testing.T) {
	f := writePythonFixture(t, "service.py", pythonFixture)
	result, err := ParsePythonFile(f)
	if err != nil {
		t.Fatalf("ParsePythonFile error: %v", err)
	}

	names := map[string]string{}
	for _, s := range result.Symbols {
		names[s.Name] = s.Kind
	}

	consts := []string{"MAX_RETRIES", "DEFAULT_TIMEOUT", "API_BASE_URL"}
	for _, c := range consts {
		if kind, ok := names[c]; !ok {
			t.Errorf("expected constant %q not found", c)
		} else if kind != KindConst {
			t.Errorf("%q: expected kind=%q, got %q", c, KindConst, kind)
		}
	}

	// default_config should be var, not const
	if kind, ok := names["default_config"]; !ok {
		t.Error("expected variable 'default_config' not found")
	} else if kind != KindVar {
		t.Errorf("default_config: expected kind=%q, got %q", KindVar, kind)
	}
}

func TestParsePythonFile_Docstrings(t *testing.T) {
	f := writePythonFixture(t, "service.py", pythonFixture)
	result, err := ParsePythonFile(f)
	if err != nil {
		t.Fatalf("ParsePythonFile error: %v", err)
	}

	for _, s := range result.Symbols {
		switch s.Name {
		case "UserService":
			if s.DocComment == "" {
				t.Error("UserService should have a doc comment")
			}
		case "process_order":
			if s.DocComment == "" {
				t.Error("process_order should have a doc comment")
			}
		case "get_user":
			if s.DocComment == "" {
				t.Error("get_user should have a doc comment")
			}
		}
	}
}

func TestParsePythonFile_Signatures(t *testing.T) {
	f := writePythonFixture(t, "service.py", pythonFixture)
	result, err := ParsePythonFile(f)
	if err != nil {
		t.Fatalf("ParsePythonFile error: %v", err)
	}

	for _, s := range result.Symbols {
		if s.Name == "UserService" && s.Kind == KindClass {
			if s.Signature == "" {
				t.Error("UserService should have a signature")
			}
			if s.Signature != "class UserService(BaseService)" {
				t.Errorf("UserService signature: expected 'class UserService(BaseService)', got %q", s.Signature)
			}
		}
		if s.Name == "get_user" {
			if s.Signature == "" {
				t.Error("get_user should have a signature")
			}
		}
	}
}

func TestParsePythonFile_LineNumbers(t *testing.T) {
	f := writePythonFixture(t, "service.py", pythonFixture)
	result, err := ParsePythonFile(f)
	if err != nil {
		t.Fatalf("ParsePythonFile error: %v", err)
	}

	for _, s := range result.Symbols {
		if s.Line <= 0 {
			t.Errorf("symbol %q has invalid line number %d", s.Name, s.Line)
		}
	}
}

func TestParsePythonFile_FlaskEndpoints(t *testing.T) {
	f := writePythonFixture(t, "app.py", pythonFlaskFixture)
	result, err := ParsePythonFile(f)
	if err != nil {
		t.Fatalf("ParsePythonFile error: %v", err)
	}

	if len(result.Endpoints) == 0 {
		t.Fatal("expected Flask endpoints, got none")
	}

	endpointMap := map[string][]string{}
	for _, ep := range result.Endpoints {
		endpointMap[ep.Path] = append(endpointMap[ep.Path], ep.Method)
	}

	if methods, ok := endpointMap["/users"]; !ok {
		t.Error("expected endpoint /users")
	} else if len(methods) < 1 {
		t.Error("/users should have at least 1 method")
	}

	if _, ok := endpointMap["/health"]; !ok {
		t.Error("expected endpoint /health")
	}
}

func TestParsePythonFile_FastAPIEndpoints(t *testing.T) {
	f := writePythonFixture(t, "main.py", pythonFastAPIFixture)
	result, err := ParsePythonFile(f)
	if err != nil {
		t.Fatalf("ParsePythonFile error: %v", err)
	}

	if len(result.Endpoints) == 0 {
		t.Fatal("expected FastAPI endpoints, got none")
	}

	// Check by handler name since paths can repeat
	handlerMethods := map[string]string{}
	for _, ep := range result.Endpoints {
		handlerMethods[ep.HandlerName] = ep.Method
	}

	if method, ok := handlerMethods["get_item"]; !ok {
		t.Error("expected endpoint handler 'get_item'")
	} else if method != "GET" {
		t.Errorf("get_item: expected method GET, got %q", method)
	}

	if method, ok := handlerMethods["create_item"]; !ok {
		t.Error("expected endpoint handler 'create_item'")
	} else if method != "POST" {
		t.Errorf("create_item: expected method POST, got %q", method)
	}

	if method, ok := handlerMethods["delete_item"]; !ok {
		t.Error("expected endpoint handler 'delete_item'")
	} else if method != "DELETE" {
		t.Errorf("delete_item: expected method DELETE, got %q", method)
	}
}

func TestParsePythonFile_NoDunderConstants(t *testing.T) {
	f := writePythonFixture(t, "service.py", pythonFixture)
	result, err := ParsePythonFile(f)
	if err != nil {
		t.Fatalf("ParsePythonFile error: %v", err)
	}

	for _, s := range result.Symbols {
		if s.Name == "__version__" {
			t.Error("__version__ dunder should not be extracted as symbol")
		}
	}
}

func TestParsePythonFile_InvalidFile(t *testing.T) {
	_, err := ParsePythonFile("/nonexistent/file.py")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestParsePythonFile_QualifiedNames(t *testing.T) {
	f := writePythonFixture(t, "service.py", pythonFixture)
	result, err := ParsePythonFile(f)
	if err != nil {
		t.Fatalf("ParsePythonFile error: %v", err)
	}

	qualNames := map[string]string{}
	for _, s := range result.Symbols {
		qualNames[s.Name] = s.QualifiedName
	}

	// Method should have qualified name with class
	if qn := qualNames["get_user"]; qn == "" {
		t.Error("get_user should have qualified name")
	} else if qn != "service.(UserService).get_user" {
		// pkg is derived from parent directory, file is in tmpdir
		// just check it contains the class
		if !contains(qn, "(UserService).get_user") {
			t.Errorf("get_user qualified name should contain (UserService), got %q", qn)
		}
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstr(s, substr))
}

func containsSubstr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
