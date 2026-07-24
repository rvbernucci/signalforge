package intelligenceaudit

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/rvbernucci/signalforge/internal/benchmark"
)

func TestStoreCapturesSafeMetadataAndProtectsBodies(t *testing.T) {
	now := time.Date(2026, 7, 23, 12, 0, 0, 0, time.UTC)
	var logs bytes.Buffer
	store, err := NewStore(Config{
		Directory: t.TempDir(), Enabled: true, Token: "operator-token",
		TTL: time.Hour, Now: func() time.Time { return now }, LogWriter: &logs,
	})
	if err != nil {
		t.Fatal(err)
	}
	recorder, err := store.Begin(context.Background(), "run-safe", "request-safe", "Compare companies. api_key: secret-value")
	if err != nil {
		t.Fatal(err)
	}
	recorder.ObserveModelCall(context.Background(), "request-interpreter/v1", "local-rocm", benchmark.Request{
		Model: "gemma", Messages: []benchmark.Message{
			{Role: "system", Content: "Interpret the request."},
			{Role: "user", Content: "Authorization: Bearer private-token"},
		},
		MaxTokens: 100, ResponseFormat: map[string]any{"type": "json_schema"},
	}, benchmark.Completion{
		Answer:    `{"intent":"company_comparison"}`,
		Usage:     benchmark.Usage{PromptTokens: 20, CompletionTokens: 5},
		StartedAt: now, Duration: time.Second, TTFT: 100 * time.Millisecond,
		FinishReason: "stop",
	}, nil)
	if err := recorder.Complete(ProjectionInput{
		RunID: "run-safe", RequestID: "request-safe", CompletedAt: now.Add(time.Second),
		Status: "completed",
	}); err != nil {
		t.Fatal(err)
	}

	public, err := store.Public("run-safe")
	if err != nil {
		t.Fatal(err)
	}
	if len(public.ModelCalls) != 1 || public.ModelCalls[0].InputTokens != 20 ||
		public.ModelCalls[0].ProtectedInputID == "" {
		t.Fatalf("public record = %+v", public)
	}
	encodedPublic, _ := os.ReadFile(filepath.Join(store.config.Directory, "run-safe.metadata.json"))
	if strings.Contains(string(encodedPublic), "private-token") || strings.Contains(string(encodedPublic), "secret-value") {
		t.Fatal("public metadata leaked a protected value")
	}
	if _, err := store.Protected("run-safe", "wrong"); err == nil {
		t.Fatal("wrong token accessed protected audit")
	}
	protected, err := store.Protected("run-safe", "operator-token")
	if err != nil {
		t.Fatal(err)
	}
	encoded := protected.ModelCalls[0].Messages[1].Content
	if strings.Contains(encoded, "private-token") || !strings.Contains(encoded, "[REDACTED]") {
		t.Fatalf("protected content was not redacted: %q", encoded)
	}
	if strings.Contains(protected.Question, "secret-value") {
		t.Fatalf("question was not redacted: %q", protected.Question)
	}
	logOutput := logs.String()
	for _, blocked := range []string{"private-token", "secret-value", "Compare companies.", `{"intent"`} {
		if strings.Contains(logOutput, blocked) {
			t.Fatalf("structured log leaked %q: %s", blocked, logOutput)
		}
	}
	if !strings.Contains(logOutput, `"event":"journey.completed"`) ||
		!strings.Contains(logOutput, `"event":"model.completed"`) {
		t.Fatalf("safe lifecycle logs were not emitted: %s", logOutput)
	}
}

func TestStoreExpiryAndPurgeAreFailClosed(t *testing.T) {
	now := time.Date(2026, 7, 23, 12, 0, 0, 0, time.UTC)
	store, err := NewStore(Config{
		Directory: t.TempDir(), Enabled: true, Token: "operator-token",
		TTL: time.Minute, Now: func() time.Time { return now },
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Begin(context.Background(), "run-expire", "request-expire", "question"); err != nil {
		t.Fatal(err)
	}
	now = now.Add(2 * time.Minute)
	if _, err := store.Protected("run-expire", "operator-token"); err == nil {
		t.Fatal("expired protected record remained readable")
	}
	public, err := store.Public("run-expire")
	if err != nil {
		t.Fatal(err)
	}
	if public.Capture.Status != "expired" || public.Capture.Available {
		t.Fatalf("capture state = %+v", public.Capture)
	}

	now = now.Add(time.Minute)
	if _, err := store.Begin(context.Background(), "run-purge", "request-purge", "question"); err != nil {
		t.Fatal(err)
	}
	if err := store.Purge("run-purge", "operator-token"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Protected("run-purge", "operator-token"); err == nil {
		t.Fatal("purged protected record remained readable")
	}
}

func TestDisabledCaptureStillRecordsMetadata(t *testing.T) {
	store, err := NewStore(Config{Directory: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	recorder, err := store.Begin(context.Background(), "run-disabled", "request-disabled", "question")
	if err != nil {
		t.Fatal(err)
	}
	recorder.ObserveModelCall(context.Background(), "final-research-analyst/v1", "local-rocm",
		benchmark.Request{Model: "gemma", Messages: []benchmark.Message{{Role: "system", Content: "answer"}}, MaxTokens: 10},
		benchmark.Completion{}, errors.New("provider failure"))
	public, err := store.Public("run-disabled")
	if err != nil {
		t.Fatal(err)
	}
	if public.Capture.Status != "disabled" || len(public.ModelCalls) != 1 ||
		public.ModelCalls[0].FailureCode != "model_call_failed" {
		t.Fatalf("public record = %+v", public)
	}
}
