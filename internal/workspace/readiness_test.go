package workspace

import (
	"testing"
	"time"

	"github.com/rvbernucci/signalforge/internal/golden"
)

func TestReadinessIdentitiesBindRuntimeModelConfigurationAndData(t *testing.T) {
	config := ServerConfig{
		Mode: ModeLive, BuildVersion: "commit-123",
		ApplicationIdentity: "sha256:application",
		RuntimeIdentity:     "sha256:runtime", ModelIdentity: "sha256:model",
		RunTimeout: 2 * time.Minute, MaxBodyBytes: 4096,
		Golden: golden.RunConfig{
			Model: "signalforge-gemma4-26b-q4", ContextConcurrency: 4,
		},
	}
	payloads := map[string][]byte{"catalog": []byte(`{"companies":20}`), "fixture": []byte(`{"case":"safe"}`)}
	first, err := buildReadinessIdentities(config, "", payloads)
	if err != nil {
		t.Fatal(err)
	}
	if first.Source != "commit-123" ||
		first.Application != "sha256:application" ||
		first.Runtime != "sha256:runtime" ||
		first.Model != "sha256:model" ||
		first.ServedModel != "signalforge-gemma4-26b-q4" {
		t.Fatalf("readiness identity binding is incomplete: %+v", first)
	}
	reordered, err := buildReadinessIdentities(
		config,
		"",
		map[string][]byte{"fixture": []byte(`{"case":"safe"}`), "catalog": []byte(`{"companies":20}`)},
	)
	if err != nil {
		t.Fatal(err)
	}
	if reordered != first {
		t.Fatal("readiness identity changed under map-order permutation")
	}

	changedData, err := buildReadinessIdentities(
		config,
		"",
		map[string][]byte{"fixture": []byte(`{"case":"safe"}`), "catalog": []byte(`{"companies":19}`)},
	)
	if err != nil {
		t.Fatal(err)
	}
	if changedData.DataSHA256 == first.DataSHA256 ||
		changedData.ConfigurationSHA256 == first.ConfigurationSHA256 {
		t.Fatal("data mutation did not change readiness identity")
	}

	config.RuntimeIdentity = "sha256:other-runtime"
	changedRuntime, err := buildReadinessIdentities(config, "", payloads)
	if err != nil {
		t.Fatal(err)
	}
	if changedRuntime.DataSHA256 != first.DataSHA256 ||
		changedRuntime.ConfigurationSHA256 == first.ConfigurationSHA256 {
		t.Fatal("runtime mutation was not isolated to configuration identity")
	}
}
