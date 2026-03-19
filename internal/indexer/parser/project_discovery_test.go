package parser

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseGoMod(t *testing.T) {
	dir := t.TempDir()
	content := `module github.com/example/app

go 1.21

require (
	google.golang.org/grpc v1.64.0
	github.com/segmentio/kafka-go v0.4.47
)

require github.com/some/indirect v1.0.0 // indirect
`
	f := filepath.Join(dir, "go.mod")
	os.WriteFile(f, []byte(content), 0644)

	mods, err := parseGoMod(f)
	if err != nil {
		t.Fatal(err)
	}

	want := map[string]bool{
		"google.golang.org/grpc":      true,
		"github.com/segmentio/kafka-go": true,
		"github.com/some/indirect":    true, // indirect not filtered at parse level
	}
	for _, m := range mods {
		if !want[m] {
			t.Errorf("unexpected module %q", m)
		}
		delete(want, m)
	}
	for m := range want {
		t.Errorf("missing module %q", m)
	}
}

func TestParseGoMod_SingleLine(t *testing.T) {
	dir := t.TempDir()
	content := `module github.com/example/app

go 1.21

require google.golang.org/grpc v1.64.0
`
	f := filepath.Join(dir, "go.mod")
	os.WriteFile(f, []byte(content), 0644)

	mods, err := parseGoMod(f)
	if err != nil {
		t.Fatal(err)
	}

	if len(mods) != 1 || mods[0] != "google.golang.org/grpc" {
		t.Errorf("got %v, want [google.golang.org/grpc]", mods)
	}
}

func TestParsePackageJSON(t *testing.T) {
	dir := t.TempDir()
	content := `{
		"dependencies": { "kafkajs": "^2.0.0", "axios": "^1.0" },
		"devDependencies": { "@grpc/grpc-js": "^1.0" }
	}`
	f := filepath.Join(dir, "package.json")
	os.WriteFile(f, []byte(content), 0644)

	pkgs, err := parsePackageJSON(f)
	if err != nil {
		t.Fatal(err)
	}

	want := map[string]bool{"kafkajs": true, "axios": true, "@grpc/grpc-js": true}
	for _, p := range pkgs {
		delete(want, p)
	}
	if len(want) > 0 {
		t.Errorf("missing packages: %v", want)
	}
	if len(pkgs) != 3 {
		t.Errorf("expected 3 packages, got %d", len(pkgs))
	}
}

func TestParseRequirementsTxt(t *testing.T) {
	dir := t.TempDir()
	content := `grpcio==1.62.0
kafka-python>=2.0.0
httpx  # http client
# comment line
-r other.txt
git+https://github.com/foo/bar.git
requests~=2.28
`
	f := filepath.Join(dir, "requirements.txt")
	os.WriteFile(f, []byte(content), 0644)

	pkgs, err := parseRequirementsTxt(f)
	if err != nil {
		t.Fatal(err)
	}

	want := map[string]bool{"grpcio": true, "kafka-python": true, "httpx": true, "requests": true}
	for _, p := range pkgs {
		if !want[p] {
			t.Errorf("unexpected package %q", p)
		}
		delete(want, p)
	}
	for p := range want {
		t.Errorf("missing package %q", p)
	}
}

func TestParsePomXML(t *testing.T) {
	dir := t.TempDir()
	content := `<project>
  <dependencies>
    <dependency>
      <groupId>io.grpc</groupId>
      <artifactId>grpc-stub</artifactId>
      <version>1.57.0</version>
    </dependency>
    <dependency>
      <groupId>org.springframework.kafka</groupId>
      <artifactId>spring-kafka</artifactId>
    </dependency>
  </dependencies>
</project>`
	f := filepath.Join(dir, "pom.xml")
	os.WriteFile(f, []byte(content), 0644)

	pkgs, err := parsePomXML(f)
	if err != nil {
		t.Fatal(err)
	}

	want := map[string]bool{
		"io.grpc:grpc-stub":                        true,
		"org.springframework.kafka:spring-kafka": true,
	}
	for _, p := range pkgs {
		if !want[p] {
			t.Errorf("unexpected package %q", p)
		}
		delete(want, p)
	}
	for p := range want {
		t.Errorf("missing package %q", p)
	}
}

func TestParseTsConfig_aliases(t *testing.T) {
	dir := t.TempDir()
	content := `{
		"compilerOptions": {
			"paths": {
				"@/*": ["src/*"],
				"@components/*": ["src/components/*"]
			}
		}
	}`
	f := filepath.Join(dir, "tsconfig.json")
	os.WriteFile(f, []byte(content), 0644)

	aliases, err := parseTsConfig(dir)
	if err != nil {
		t.Fatal(err)
	}

	if aliases["@/"] != "src/" {
		t.Errorf("expected @/ → src/, got %q", aliases["@/"])
	}
	if aliases["@components/"] != "src/components/" {
		t.Errorf("expected @components/ → src/components/, got %q", aliases["@components/"])
	}
}

func TestParseTsConfig_fallback(t *testing.T) {
	dir := t.TempDir()
	// tsconfig.json has no paths, tsconfig.app.json has paths
	os.WriteFile(filepath.Join(dir, "tsconfig.json"), []byte(`{"compilerOptions":{}}`), 0644)
	os.WriteFile(filepath.Join(dir, "tsconfig.app.json"), []byte(`{
		"compilerOptions": {"paths": {"@/*": ["src/*"]}}
	}`), 0644)

	aliases, err := parseTsConfig(dir)
	if err != nil {
		t.Fatal(err)
	}
	if aliases["@/"] != "src/" {
		t.Errorf("expected @/ → src/ from fallback, got %q", aliases["@/"])
	}
}

func TestParseTsConfig_missing(t *testing.T) {
	dir := t.TempDir()

	aliases, err := parseTsConfig(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(aliases) != 0 {
		t.Errorf("expected empty aliases, got %v", aliases)
	}
}
