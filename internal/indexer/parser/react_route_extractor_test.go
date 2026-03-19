package parser

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/xdotech/goatlas/internal/indexer/domain"
)

const reactNavFixture = `import { createNativeStackNavigator } from '@react-navigation/native-stack'

const Stack = createNativeStackNavigator()

export function AppNavigator() {
  return (
    <Stack.Navigator>
      <Stack.Screen name="Home" component={HomeScreen} />
      <Stack.Screen name="Profile" component={ProfileScreen} />
      <Stack.Screen name="OrderDetail" component={OrderDetailScreen} />
    </Stack.Navigator>
  )
}
`

const reactRouterFixture = `import { createBrowserRouter } from 'react-router-dom'

const router = createBrowserRouter([
  { path: "/", element: <RootLayout /> },
  { path: "/login", element: <LoginPage /> },
  { path: "/dashboard", element: <Dashboard /> },
])

export function AppRoutes() {
  return (
    <Routes>
      <Route path="/users" element={<UserList />} />
      <Route path="/users/:id" element={<UserDetail />} />
    </Routes>
  )
}
`

const noReactNavFixture = `import React from 'react'

export function PlainComponent() {
  return null
}
`

func writeRouteFixture(t *testing.T, name, content string) string {
	t.Helper()
	dir := t.TempDir()
	f := filepath.Join(dir, name)
	if err := os.WriteFile(f, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	return f
}

func TestExtractReactRoutes_Navigation(t *testing.T) {
	f := writeRouteFixture(t, "AppNavigator.tsx", reactNavFixture)
	imports := []domain.Import{{ImportPath: "@react-navigation/native-stack"}}

	endpoints, err := ExtractReactRoutes(f, imports)
	if err != nil {
		t.Fatalf("ExtractReactRoutes error: %v", err)
	}

	if len(endpoints) == 0 {
		t.Fatal("expected at least one screen")
	}

	routes := map[string]string{}
	for _, ep := range endpoints {
		routes[ep.Path] = ep.HandlerName
		if ep.Framework != FrameworkReactNavigation {
			t.Errorf("expected framework %q, got %q", FrameworkReactNavigation, ep.Framework)
		}
		if ep.Method != "SCREEN" {
			t.Errorf("expected method 'SCREEN', got %q", ep.Method)
		}
	}

	expected := map[string]string{
		"Home":        "HomeScreen",
		"Profile":     "ProfileScreen",
		"OrderDetail": "OrderDetailScreen",
	}
	for name, handler := range expected {
		if got, ok := routes[name]; !ok {
			t.Errorf("expected route %q not found", name)
		} else if got != handler {
			t.Errorf("route %q: expected handler %q, got %q", name, handler, got)
		}
	}
}

func TestExtractReactRoutes_Router(t *testing.T) {
	f := writeRouteFixture(t, "routes.tsx", reactRouterFixture)
	imports := []domain.Import{{ImportPath: "react-router-dom"}}

	endpoints, err := ExtractReactRoutes(f, imports)
	if err != nil {
		t.Fatalf("ExtractReactRoutes error: %v", err)
	}

	if len(endpoints) == 0 {
		t.Fatal("expected at least one route")
	}

	paths := map[string]bool{}
	for _, ep := range endpoints {
		paths[ep.Path] = true
		if ep.Framework != FrameworkReactRouter {
			t.Errorf("expected framework %q, got %q", FrameworkReactRouter, ep.Framework)
		}
	}

	for _, p := range []string{"/", "/login", "/dashboard"} {
		if !paths[p] {
			t.Errorf("expected path %q not found", p)
		}
	}
}

func TestExtractReactRoutes_NoFramework(t *testing.T) {
	f := writeRouteFixture(t, "plain.tsx", noReactNavFixture)
	imports := []domain.Import{{ImportPath: "react"}}

	endpoints, err := ExtractReactRoutes(f, imports)
	if err != nil {
		t.Fatalf("ExtractReactRoutes error: %v", err)
	}
	if len(endpoints) != 0 {
		t.Errorf("expected 0 endpoints for plain React, got %d", len(endpoints))
	}
}

func TestDetectReactFramework(t *testing.T) {
	cases := []struct {
		imports  []domain.Import
		expected string
	}{
		{[]domain.Import{{ImportPath: "@react-navigation/native"}}, FrameworkReactNavigation},
		{[]domain.Import{{ImportPath: "@react-navigation/native-stack"}}, FrameworkReactNavigation},
		{[]domain.Import{{ImportPath: "react-router-dom"}}, FrameworkReactRouter},
		{[]domain.Import{{ImportPath: "react-router"}}, FrameworkReactRouter},
		{[]domain.Import{{ImportPath: "expo-router"}}, FrameworkExpoRouter},
		{[]domain.Import{{ImportPath: "react"}}, ""},
	}

	for _, tc := range cases {
		got := detectReactFramework(tc.imports)
		if got != tc.expected {
			t.Errorf("detectReactFramework(%v) = %q, want %q", tc.imports, got, tc.expected)
		}
	}
}

func TestExpoRouteFromPath(t *testing.T) {
	cases := []struct {
		path     string
		expected string
	}{
		{"/project/app/index.tsx", "/"},
		{"/project/app/about.tsx", "/about"},
		{"/project/app/(tabs)/profile.tsx", "/(tabs)/profile"},
		{"/project/app/users/[id].tsx", "/users/[id]"},
	}

	for _, tc := range cases {
		// Create a temp file at the given path structure
		dir := t.TempDir()
		f := filepath.Join(dir, tc.path)
		os.MkdirAll(filepath.Dir(f), 0700)
		os.WriteFile(f, []byte(""), 0600)

		got := expoRouteFromPath(tc.path)
		if got != tc.expected {
			t.Errorf("expoRouteFromPath(%q) = %q, want %q", tc.path, got, tc.expected)
		}
	}
}
