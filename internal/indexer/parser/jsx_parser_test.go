package parser

import (
	"os"
	"path/filepath"
	"testing"
)

const tsxFixture = `import React, { useState, useEffect } from 'react'
import { View, Text } from 'react-native'
import axios from 'axios'
import type { FC } from 'react'

// UserCard displays a single user.
export function UserCard({ name }: { name: string }) {
  return <View><Text>{name}</Text></View>
}

export const ProfileScreen: React.FC<{ userId: string }> = ({ userId }) => {
  return <View />
}

export const OrderList = () => {
  return <View />
}

// useUserData fetches user data from the API.
export function useUserData(id: string) {
  const [data, setData] = useState(null)
  useEffect(() => {
    axios.get('/api/users/' + id).then(r => setData(r.data))
  }, [id])
  return data
}

export const useAuth = () => {
  return { user: null, login: () => {} }
}

export interface UserProfile {
  id: string
  name: string
  email: string
}

export type OrderStatus = 'pending' | 'shipped' | 'delivered'

export type ApiResponse<T> = {
  data: T
  error: string | null
}

// Not a component — lowercase
function helperUtil() {}
const internalHelper = () => {}
`

const classComponentFixture = `import React, { Component } from 'react'

export class LegacyWidget extends React.Component {
  render() { return null }
}

export class AnotherWidget extends Component {
  render() { return null }
}
`

func writeJSXFixture(t *testing.T, name, content string) string {
	t.Helper()
	dir := t.TempDir()
	f := filepath.Join(dir, name)
	if err := os.WriteFile(f, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	return f
}

func TestParseJSXFile_Components(t *testing.T) {
	f := writeJSXFixture(t, "Screen.tsx", tsxFixture)
	result, err := ParseJSXFile(f)
	if err != nil {
		t.Fatalf("ParseJSXFile error: %v", err)
	}

	names := map[string]string{}
	for _, s := range result.Symbols {
		names[s.Name] = s.Kind
	}

	components := []string{"UserCard", "ProfileScreen", "OrderList"}
	for _, c := range components {
		if kind, ok := names[c]; !ok {
			t.Errorf("expected component %q not found", c)
		} else if kind != KindComponent {
			t.Errorf("%q: expected kind=%q, got %q", c, KindComponent, kind)
		}
	}
}

func TestParseJSXFile_Hooks(t *testing.T) {
	f := writeJSXFixture(t, "hooks.tsx", tsxFixture)
	result, err := ParseJSXFile(f)
	if err != nil {
		t.Fatalf("ParseJSXFile error: %v", err)
	}

	names := map[string]string{}
	for _, s := range result.Symbols {
		names[s.Name] = s.Kind
	}

	hooks := []string{"useUserData", "useAuth"}
	for _, h := range hooks {
		if kind, ok := names[h]; !ok {
			t.Errorf("expected hook %q not found", h)
		} else if kind != KindHook {
			t.Errorf("%q: expected kind=%q, got %q", h, KindHook, kind)
		}
	}
}

func TestParseJSXFile_Interfaces(t *testing.T) {
	f := writeJSXFixture(t, "types.tsx", tsxFixture)
	result, err := ParseJSXFile(f)
	if err != nil {
		t.Fatalf("ParseJSXFile error: %v", err)
	}

	names := map[string]string{}
	for _, s := range result.Symbols {
		names[s.Name] = s.Kind
	}

	if kind, ok := names["UserProfile"]; !ok {
		t.Error("expected interface 'UserProfile' not found")
	} else if kind != KindInterface {
		t.Errorf("UserProfile: expected kind=%q, got %q", KindInterface, kind)
	}
}

func TestParseJSXFile_TypeAliases(t *testing.T) {
	f := writeJSXFixture(t, "types.tsx", tsxFixture)
	result, err := ParseJSXFile(f)
	if err != nil {
		t.Fatalf("ParseJSXFile error: %v", err)
	}

	names := map[string]string{}
	for _, s := range result.Symbols {
		names[s.Name] = s.Kind
	}

	for _, typeName := range []string{"OrderStatus", "ApiResponse"} {
		if kind, ok := names[typeName]; !ok {
			t.Errorf("expected type alias %q not found", typeName)
		} else if kind != KindTypeAlias {
			t.Errorf("%q: expected kind=%q, got %q", typeName, KindTypeAlias, kind)
		}
	}
}

func TestParseJSXFile_NoLowercaseSymbols(t *testing.T) {
	f := writeJSXFixture(t, "util.tsx", tsxFixture)
	result, err := ParseJSXFile(f)
	if err != nil {
		t.Fatalf("ParseJSXFile error: %v", err)
	}

	for _, s := range result.Symbols {
		if s.Name == "helperUtil" || s.Name == "internalHelper" {
			t.Errorf("lowercase symbol %q should not be extracted", s.Name)
		}
	}
}

func TestParseJSXFile_Imports(t *testing.T) {
	f := writeJSXFixture(t, "Screen.tsx", tsxFixture)
	result, err := ParseJSXFile(f)
	if err != nil {
		t.Fatalf("ParseJSXFile error: %v", err)
	}

	paths := map[string]bool{}
	for _, imp := range result.Imports {
		paths[imp.ImportPath] = true
	}

	expected := []string{"react", "react-native", "axios"}
	for _, p := range expected {
		if !paths[p] {
			t.Errorf("expected import %q not found", p)
		}
	}
}

func TestParseJSXFile_ClassComponents(t *testing.T) {
	f := writeJSXFixture(t, "legacy.tsx", classComponentFixture)
	result, err := ParseJSXFile(f)
	if err != nil {
		t.Fatalf("ParseJSXFile error: %v", err)
	}

	names := map[string]string{}
	for _, s := range result.Symbols {
		names[s.Name] = s.Kind
	}

	for _, name := range []string{"LegacyWidget", "AnotherWidget"} {
		if kind, ok := names[name]; !ok {
			t.Errorf("expected class component %q not found", name)
		} else if kind != KindComponent {
			t.Errorf("%q: expected kind=%q, got %q", name, KindComponent, kind)
		}
	}
}

func TestParseJSXFile_DocComment(t *testing.T) {
	f := writeJSXFixture(t, "Screen.tsx", tsxFixture)
	result, err := ParseJSXFile(f)
	if err != nil {
		t.Fatalf("ParseJSXFile error: %v", err)
	}

	for _, s := range result.Symbols {
		if s.Name == "UserCard" && s.DocComment == "" {
			t.Error("UserCard should have a doc comment")
		}
		if s.Name == "useUserData" && s.DocComment == "" {
			t.Error("useUserData should have a doc comment")
		}
	}
}

func TestParseJSXFile_LineNumbers(t *testing.T) {
	f := writeJSXFixture(t, "Screen.tsx", tsxFixture)
	result, err := ParseJSXFile(f)
	if err != nil {
		t.Fatalf("ParseJSXFile error: %v", err)
	}

	for _, s := range result.Symbols {
		if s.Line <= 0 {
			t.Errorf("symbol %q has invalid line number %d", s.Name, s.Line)
		}
	}
}

func TestParseJSXFile_InvalidFile(t *testing.T) {
	_, err := ParseJSXFile("/nonexistent/file.tsx")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestModuleNameFromPath(t *testing.T) {
	cases := []struct {
		path     string
		expected string
	}{
		{"src/screens/HomeScreen.tsx", "screens"},
		{"components/Button.tsx", "components"},
		{"App.tsx", "App"},
	}

	for _, tc := range cases {
		got := moduleNameFromPath(tc.path)
		if got != tc.expected {
			t.Errorf("moduleNameFromPath(%q) = %q, want %q", tc.path, got, tc.expected)
		}
	}
}
