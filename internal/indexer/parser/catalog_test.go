package parser

import "testing"

func hasGoConnType(patterns []GoCallPattern, connType string) bool {
	for _, p := range patterns {
		if p.ConnType == connType {
			return true
		}
	}
	return false
}

func TestLookupCatalog_grpc(t *testing.T) {
	deps := &ProjectDeps{GoModules: []string{"google.golang.org/grpc"}}
	cfg := LookupCatalog(deps)

	if len(cfg.Go) == 0 {
		t.Fatal("expected Go patterns for google.golang.org/grpc")
	}
	if !hasGoConnType(cfg.Go, "grpc") {
		t.Error("expected pattern with conn_type=grpc")
	}
}

func TestLookupCatalog_kafka(t *testing.T) {
	deps := &ProjectDeps{GoModules: []string{"github.com/segmentio/kafka-go"}}
	cfg := LookupCatalog(deps)

	if !hasGoConnType(cfg.Go, "kafka_consume") {
		t.Error("expected kafka_consume Go pattern")
	}
	if !hasGoConnType(cfg.Go, "kafka_publish") {
		t.Error("expected kafka_publish Go pattern")
	}
}

func TestLookupCatalog_prefix(t *testing.T) {
	deps := &ProjectDeps{GoModules: []string{"google.golang.org/grpc/v2"}}
	cfg := LookupCatalog(deps)

	if !hasGoConnType(cfg.Go, "grpc") {
		t.Error("expected grpc patterns from prefix match on google.golang.org/grpc/v2")
	}
}

func TestLookupCatalog_npm(t *testing.T) {
	deps := &ProjectDeps{NPMPkgs: []string{"kafkajs", "axios"}}
	cfg := LookupCatalog(deps)

	if len(cfg.TypeScript) == 0 {
		t.Error("expected TypeScript patterns for kafkajs/axios")
	}
}

func TestLookupCatalog_python(t *testing.T) {
	deps := &ProjectDeps{PyPkgs: []string{"kafka-python", "grpcio"}}
	cfg := LookupCatalog(deps)

	if len(cfg.Python) == 0 {
		t.Fatal("expected Python patterns for kafka-python and grpcio")
	}
	hasKafka, hasGRPC := false, false
	for _, p := range cfg.Python {
		if p.ConnType == "kafka_consume" {
			hasKafka = true
		}
		if p.ConnType == "grpc" {
			hasGRPC = true
		}
	}
	if !hasKafka {
		t.Error("expected kafka_consume Python pattern")
	}
	if !hasGRPC {
		t.Error("expected grpc Python pattern")
	}
}

func TestLookupCatalog_maven(t *testing.T) {
	deps := &ProjectDeps{MavenPkgs: []string{"io.grpc:grpc-stub"}}
	cfg := LookupCatalog(deps)

	if len(cfg.Java) == 0 {
		t.Fatal("expected Java patterns for io.grpc:grpc-stub")
	}
	hasGRPC := false
	for _, p := range cfg.Java {
		if p.ConnType == "grpc" {
			hasGRPC = true
		}
	}
	if !hasGRPC {
		t.Error("expected grpc Java pattern")
	}
}

func TestLookupCatalog_empty(t *testing.T) {
	deps := &ProjectDeps{}
	cfg := LookupCatalog(deps)

	if len(cfg.Go) > 0 || len(cfg.TypeScript) > 0 {
		t.Error("expected empty patterns for empty deps")
	}
}
