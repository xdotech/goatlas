package parser

import "testing"

func TestDetectFrameworkFromPath_django(t *testing.T) {
	hint := DetectFrameworkFromPath("/app/myservice/views.py")
	if hint == nil || hint.Framework != "django" {
		t.Errorf("expected django, got %v", hint)
	}
	if hint.EntryPointMultiplier != 3.0 {
		t.Errorf("expected 3.0 multiplier, got %f", hint.EntryPointMultiplier)
	}
}

func TestDetectFrameworkFromPath_djangoUrls(t *testing.T) {
	hint := DetectFrameworkFromPath("/app/urls.py")
	if hint == nil || hint.Framework != "django" {
		t.Errorf("expected django, got %v", hint)
	}
	if hint.EntryPointMultiplier != 2.0 {
		t.Errorf("expected 2.0 multiplier, got %f", hint.EntryPointMultiplier)
	}
}

func TestDetectFrameworkFromPath_fastapi(t *testing.T) {
	hint := DetectFrameworkFromPath("/app/routers/orders.py")
	if hint == nil || hint.Framework != "fastapi" {
		t.Errorf("expected fastapi, got %v", hint)
	}
	if hint.EntryPointMultiplier != 2.5 {
		t.Errorf("expected 2.5 multiplier, got %f", hint.EntryPointMultiplier)
	}
}

func TestDetectFrameworkFromPath_fastapiEndpoints(t *testing.T) {
	hint := DetectFrameworkFromPath("/app/endpoints/users.py")
	if hint == nil || hint.Framework != "fastapi" {
		t.Errorf("expected fastapi, got %v", hint)
	}
}

func TestDetectFrameworkFromPath_flask(t *testing.T) {
	hint := DetectFrameworkFromPath("/app/routes/payments.py")
	if hint == nil || hint.Framework != "flask" {
		t.Errorf("expected flask, got %v", hint)
	}
	if hint.EntryPointMultiplier != 2.5 {
		t.Errorf("expected 2.5 multiplier, got %f", hint.EntryPointMultiplier)
	}
}

func TestDetectFrameworkFromPath_none(t *testing.T) {
	hint := DetectFrameworkFromPath("/app/utils.py")
	if hint != nil {
		t.Errorf("expected nil for utils.py, got %v", hint)
	}
}

func TestDetectFrameworkFromPath_initSkipped(t *testing.T) {
	// __init__.py in routers should not trigger fastapi
	hint := DetectFrameworkFromPath("/app/routers/__init__.py")
	if hint != nil {
		t.Errorf("expected nil for __init__.py, got %v", hint)
	}
}

func TestDetectFrameworkFromPath_spring(t *testing.T) {
	hint := DetectFrameworkFromPath("/app/controller/UserController.java")
	if hint == nil || hint.Framework != "spring" {
		t.Errorf("expected spring, got %v", hint)
	}
	if hint.EntryPointMultiplier != 3.0 {
		t.Errorf("expected 3.0 multiplier, got %f", hint.EntryPointMultiplier)
	}
}

func TestDetectFrameworkFromPath_jaxrs(t *testing.T) {
	hint := DetectFrameworkFromPath("/app/resource/OrderResource.java")
	if hint == nil || hint.Framework != "jax-rs" {
		t.Errorf("expected jax-rs, got %v", hint)
	}
	if hint.EntryPointMultiplier != 3.0 {
		t.Errorf("expected 3.0 multiplier, got %f", hint.EntryPointMultiplier)
	}
}
