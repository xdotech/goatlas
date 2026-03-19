package parser

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectJavaConnections_grpc(t *testing.T) {
	dir := t.TempDir()
	src := `package com.example;

import io.grpc.ManagedChannel;
import io.grpc.ManagedChannelBuilder;

public class GrpcClient {
    void connect() {
        ManagedChannel channel = ManagedChannelBuilder.forAddress("payment-svc", 9090).build();
    }
}
`
	f := filepath.Join(dir, "GrpcClient.java")
	os.WriteFile(f, []byte(src), 0644)

	cfg := []JavaCallPattern{
		{ImportContains: "io.grpc", MethodCall: "forAddress", TargetArgIndex: 0, ConnType: "grpc"},
	}

	conns, err := DetectJavaConnections(f, cfg)
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

func TestDetectJavaConnections_kafkaListener(t *testing.T) {
	dir := t.TempDir()
	src := `package com.example;

import org.springframework.kafka.annotation.KafkaListener;

public class OrderConsumer {
    @KafkaListener(topics = "orders")
    public void consume(String msg) {}
}
`
	f := filepath.Join(dir, "OrderConsumer.java")
	os.WriteFile(f, []byte(src), 0644)

	cfg := []JavaCallPattern{
		{ImportContains: "springframework.kafka", Annotation: "KafkaListener", TargetAttribute: "topics", ConnType: "kafka_consume"},
	}

	conns, err := DetectJavaConnections(f, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if len(conns) == 0 {
		t.Fatal("expected kafka_consume connection from @KafkaListener")
	}
	if conns[0].ConnType != "kafka_consume" {
		t.Errorf("expected kafka_consume, got %q", conns[0].ConnType)
	}
	if conns[0].Target != "orders" {
		t.Errorf("expected target 'orders', got %q", conns[0].Target)
	}
}

func TestDetectJavaConnections_feignClient(t *testing.T) {
	dir := t.TempDir()
	src := `package com.example;

import org.springframework.cloud.openfeign.FeignClient;

@FeignClient(name = "product-svc")
public interface ProductClient {}
`
	f := filepath.Join(dir, "ProductClient.java")
	os.WriteFile(f, []byte(src), 0644)

	cfg := []JavaCallPattern{
		{ImportContains: "openfeign", Annotation: "FeignClient", TargetAttribute: "name", ConnType: "http_api"},
	}

	conns, err := DetectJavaConnections(f, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if len(conns) == 0 {
		t.Fatal("expected http_api connection from @FeignClient")
	}
	if conns[0].ConnType != "http_api" {
		t.Errorf("expected http_api, got %q", conns[0].ConnType)
	}
	if conns[0].Target != "product-svc" {
		t.Errorf("expected target 'product-svc', got %q", conns[0].Target)
	}
}
