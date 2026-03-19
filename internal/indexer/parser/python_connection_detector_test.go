package parser

import (
	"os"
	"path/filepath"
	"testing"

	tree_sitter "github.com/tree-sitter/go-tree-sitter"
	tree_sitter_python "github.com/tree-sitter/tree-sitter-python/bindings/go"
)

func TestDetectPythonConnections_kafka(t *testing.T) {
	dir := t.TempDir()
	src := `from kafka import KafkaConsumer
consumer = KafkaConsumer('orders', bootstrap_servers='kafka:9092')
`
	f := filepath.Join(dir, "consumer.py")
	os.WriteFile(f, []byte(src), 0644)

	cfg := []PyCallPattern{
		{ModuleContains: "kafka", CallPattern: "KafkaConsumer(", TargetArgIndex: 0, ConnType: "kafka_consume"},
	}

	conns, err := DetectPythonConnections(f, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if len(conns) == 0 {
		t.Fatal("expected kafka_consume connection")
	}
	if conns[0].ConnType != "kafka_consume" {
		t.Errorf("expected kafka_consume, got %q", conns[0].ConnType)
	}
	if conns[0].Target != "orders" {
		t.Errorf("expected target 'orders', got %q", conns[0].Target)
	}
}

func TestDetectPythonConnections_grpc(t *testing.T) {
	dir := t.TempDir()
	src := `import grpc
channel = grpc.insecure_channel('payment-svc:50051')
`
	f := filepath.Join(dir, "client.py")
	os.WriteFile(f, []byte(src), 0644)

	cfg := []PyCallPattern{
		{ModuleContains: "grpc", CallPattern: "insecure_channel(", TargetArgIndex: 0, ConnType: "grpc"},
	}

	conns, err := DetectPythonConnections(f, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if len(conns) == 0 {
		t.Fatal("expected grpc connection")
	}
	if conns[0].ConnType != "grpc" {
		t.Errorf("expected grpc, got %q", conns[0].ConnType)
	}
}

func TestDetectPythonConnections_httpx(t *testing.T) {
	dir := t.TempDir()
	src := `import httpx
client = httpx.Client(base_url='http://api.example.com')
`
	f := filepath.Join(dir, "http_client.py")
	os.WriteFile(f, []byte(src), 0644)

	cfg := []PyCallPattern{
		{ModuleContains: "httpx", CallPattern: "Client(", TargetKeyword: "base_url", ConnType: "http_api"},
	}

	conns, err := DetectPythonConnections(f, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if len(conns) == 0 {
		t.Fatal("expected http_api connection")
	}
	if conns[0].ConnType != "http_api" {
		t.Errorf("expected http_api, got %q", conns[0].ConnType)
	}
}

func TestDetectPythonConnections_noMatch(t *testing.T) {
	dir := t.TempDir()
	src := `x = 1 + 2
print("hello world")
`
	f := filepath.Join(dir, "noop.py")
	os.WriteFile(f, []byte(src), 0644)

	cfg := []PyCallPattern{
		{ModuleContains: "kafka", CallPattern: "KafkaConsumer(", ConnType: "kafka_consume"},
	}

	conns, err := DetectPythonConnections(f, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if len(conns) != 0 {
		t.Errorf("expected no connections, got %d", len(conns))
	}
}

func TestBuildPythonImportMap(t *testing.T) {
	src := []byte(`from kafka import KafkaConsumer
import grpc
import httpx as h
`)
	p := tree_sitter.NewParser()
	defer p.Close()
	lang := tree_sitter.NewLanguage(tree_sitter_python.Language())
	if err := p.SetLanguage(lang); err != nil {
		t.Skipf("tree-sitter python unavailable: %v", err)
	}
	tree := p.Parse(src, nil)
	defer tree.Close()

	imports := buildPythonImportMap(tree.RootNode(), src)

	if imports["KafkaConsumer"] != "kafka" {
		t.Errorf("expected KafkaConsumer→kafka, got %v", imports)
	}
	if imports["grpc"] != "grpc" {
		t.Errorf("expected grpc→grpc, got %v", imports)
	}
	if imports["h"] != "httpx" {
		t.Errorf("expected h→httpx, got %v", imports)
	}
}
