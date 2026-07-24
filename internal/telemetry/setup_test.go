package telemetry

import (
	"context"
	"net/http"
	"testing"
)

func TestLoadFromEnvIsOptInAndRejectsUnsafeEndpoint(t *testing.T) {
	t.Setenv("SIGNALFORGE_OTEL_ENABLED", "")
	t.Setenv("SIGNALFORGE_OTEL_INSECURE", "")
	t.Setenv("SIGNALFORGE_OTEL_ALLOW_PRIVATE_NETWORK", "")
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "")
	config, err := LoadFromEnv("test")
	if err != nil || config.Enabled {
		t.Fatalf("config = %+v, err = %v", config, err)
	}

	t.Setenv("SIGNALFORGE_OTEL_ENABLED", "true")
	t.Setenv("SIGNALFORGE_OTEL_INSECURE", "true")
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "http://collector.example:4318")
	if _, err := LoadFromEnv("test"); err == nil {
		t.Fatal("external insecure OTLP endpoint was accepted")
	}

	t.Setenv("SIGNALFORGE_OTEL_ALLOW_PRIVATE_NETWORK", "true")
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "http://alloy:4318")
	config, err = LoadFromEnv("test")
	if err != nil || !config.AllowPrivate {
		t.Fatalf("private collector config = %+v, err = %v", config, err)
	}

	t.Setenv("SIGNALFORGE_OTEL_ALLOW_PRIVATE_NETWORK", "false")
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "http://127.0.0.1:4318")
	config, err = LoadFromEnv("test")
	if err != nil || !config.Enabled || !config.Insecure {
		t.Fatalf("config = %+v, err = %v", config, err)
	}
}

func TestDisabledRuntimeIsNoop(t *testing.T) {
	runtime, err := Setup(context.Background(), Config{})
	if err != nil {
		t.Fatal(err)
	}
	handler := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})
	if runtime.WrapHTTP(handler) == nil {
		t.Fatal("disabled runtime returned a nil handler")
	}
	if err := runtime.Shutdown(context.Background()); err != nil {
		t.Fatal(err)
	}
}
