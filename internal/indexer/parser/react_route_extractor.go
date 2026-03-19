package parser

import (
	"bufio"
	"os"
	"regexp"
	"strings"

	"github.com/xdotech/goatlas/internal/indexer/domain"
)

// React framework identifiers stored in api_endpoints.framework
const (
	FrameworkReactRouter     = "react-router"
	FrameworkReactNavigation = "react-navigation"
	FrameworkExpoRouter      = "expo-router"
)

var (
	// React Navigation: <Stack.Screen name="RouteName" component={...} />
	reNavScreen = regexp.MustCompile(`<\w+\.Screen\s[^>]*name=["']([^"']+)["'][^>]*component=\{(\w+)\}`)
	// Reversed attribute order: component={X} name="Y"
	reNavScreenRev = regexp.MustCompile(`<\w+\.Screen\s[^>]*component=\{(\w+)\}[^>]*name=["']([^"']+)["']`)

	// React Router v6: <Route path="/foo" element={<Comp />} />
	reRouterRoute = regexp.MustCompile(`<Route\s[^>]*path=["']([^"']+)["'][^>]*element=\{<(\w+)`)
	// createBrowserRouter / createHashRouter object form: { path: "/foo", element: ... }
	reRouterObject = regexp.MustCompile(`path:\s*["']([^"']+)["']`)

	// Expo Router file-based — detect _layout.tsx / (tabs)/_layout.tsx patterns
	// Expo Router routes are inferred from file paths, no explicit route strings needed.
	// We detect the createExpoRouter sentinel import instead.

	// Import detection for framework identification
	reImportFrom = regexp.MustCompile(`from\s+['"](@react-navigation/[^'"]+|react-router[^'"]*|expo-router)['"](;?)`)
)

// ExtractReactRoutes detects React Router and React Navigation route definitions.
// It uses imports to identify the framework, then scans for route declarations.
func ExtractReactRoutes(filePath string, imports []domain.Import) ([]domain.APIEndpoint, error) {
	framework := detectReactFramework(imports)
	if framework == "" {
		return nil, nil
	}

	f, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var endpoints []domain.APIEndpoint
	lineNum := 0

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		switch framework {
		case FrameworkReactNavigation:
			eps := extractNavScreens(line, lineNum, framework)
			endpoints = append(endpoints, eps...)

		case FrameworkReactRouter:
			eps := extractRouterRoutes(line, lineNum, framework)
			endpoints = append(endpoints, eps...)

		case FrameworkExpoRouter:
			// Expo Router: file-path-based routing — record the file itself as a route
			if lineNum == 1 {
				routePath := expoRouteFromPath(filePath)
				endpoints = append(endpoints, domain.APIEndpoint{
					Method:    "GET",
					Path:      routePath,
					Framework: FrameworkExpoRouter,
					Line:      1,
				})
			}
		}
	}

	return endpoints, scanner.Err()
}

func detectReactFramework(imports []domain.Import) string {
	for _, imp := range imports {
		switch {
		case strings.Contains(imp.ImportPath, "@react-navigation/"):
			return FrameworkReactNavigation
		case strings.HasPrefix(imp.ImportPath, "react-router"):
			return FrameworkReactRouter
		case imp.ImportPath == "expo-router":
			return FrameworkExpoRouter
		}
	}
	return ""
}

func extractNavScreens(line string, lineNum int, framework string) []domain.APIEndpoint {
	var eps []domain.APIEndpoint

	if m := reNavScreen.FindStringSubmatch(line); m != nil {
		eps = append(eps, domain.APIEndpoint{
			Method:      "SCREEN",
			Path:        m[1],
			HandlerName: m[2],
			Framework:   framework,
			Line:        lineNum,
		})
	} else if m := reNavScreenRev.FindStringSubmatch(line); m != nil {
		eps = append(eps, domain.APIEndpoint{
			Method:      "SCREEN",
			Path:        m[2],
			HandlerName: m[1],
			Framework:   framework,
			Line:        lineNum,
		})
	}

	return eps
}

func extractRouterRoutes(line string, lineNum int, framework string) []domain.APIEndpoint {
	var eps []domain.APIEndpoint

	if m := reRouterRoute.FindStringSubmatch(line); m != nil {
		eps = append(eps, domain.APIEndpoint{
			Method:      "GET",
			Path:        m[1],
			HandlerName: m[2],
			Framework:   framework,
			Line:        lineNum,
		})
	} else if m := reRouterObject.FindStringSubmatch(line); m != nil {
		eps = append(eps, domain.APIEndpoint{
			Method:    "GET",
			Path:      m[1],
			Framework: framework,
			Line:      lineNum,
		})
	}

	return eps
}

// expoRouteFromPath converts a file path to an Expo Router URL pattern.
// e.g. app/(tabs)/profile.tsx → /(tabs)/profile
func expoRouteFromPath(filePath string) string {
	// Find "app/" directory as Expo Router root
	idx := strings.Index(filePath, "/app/")
	if idx < 0 {
		return filePath
	}
	route := filePath[idx+4:] // strip up to and including /app
	// Remove extension
	if i := strings.LastIndex(route, "."); i > 0 {
		route = route[:i]
	}
	// Strip index suffix
	route = strings.TrimSuffix(route, "/index")
	if route == "" {
		route = "/"
	}
	return route
}

// DetectReactFrameworkFromLine is used for import-line-based detection
// when parsing incrementally without pre-built import list.
func DetectReactFrameworkFromLine(line string) string {
	if m := reImportFrom.FindStringSubmatch(line); m != nil {
		imp := m[1]
		switch {
		case strings.Contains(imp, "@react-navigation/"):
			return FrameworkReactNavigation
		case strings.HasPrefix(imp, "react-router"):
			return FrameworkReactRouter
		case imp == "expo-router":
			return FrameworkExpoRouter
		}
	}
	return ""
}
