package workspace

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"sort"
	"strings"
)

const ReadinessIdentitySchemaV1 = "signalforge/readiness-identities/v1"

type ReadinessIdentities struct {
	SchemaVersion       string `json:"schema_version"`
	Source              string `json:"source"`
	Application         string `json:"application"`
	Runtime             string `json:"runtime"`
	Model               string `json:"model"`
	ServedModel         string `json:"served_model"`
	ConfigurationSHA256 string `json:"configuration_sha256"`
	DataSHA256          string `json:"data_sha256"`
}

func buildReadinessIdentities(
	config ServerConfig,
	fixtureModel string,
	dataPayloads map[string][]byte,
) (ReadinessIdentities, error) {
	source := strings.TrimSpace(config.BuildVersion)
	if source == "" {
		source = "working-tree"
	}
	application := strings.TrimSpace(config.ApplicationIdentity)
	if application == "" {
		application = "source@" + source
	}
	runtimeIdentity := strings.TrimSpace(config.RuntimeIdentity)
	if runtimeIdentity == "" {
		runtimeIdentity = "workspace-go@" + source
	}
	servedModel := strings.TrimSpace(config.Golden.Model)
	if servedModel == "" {
		servedModel = strings.TrimSpace(fixtureModel)
	}
	if servedModel == "" {
		servedModel = "not-required"
	}
	modelIdentity := strings.TrimSpace(config.ModelIdentity)
	if modelIdentity == "" {
		modelIdentity = "served-model@" + servedModel
	}
	dataSHA, err := hashNamedPayloads(dataPayloads)
	if err != nil {
		return ReadinessIdentities{}, err
	}
	configPayload, err := json.Marshal(struct {
		SchemaVersion  string `json:"schema_version"`
		Mode           string `json:"mode"`
		Source         string `json:"source"`
		Application    string `json:"application"`
		Runtime        string `json:"runtime"`
		Model          string `json:"model"`
		ServedModel    string `json:"served_model"`
		DataSHA256     string `json:"data_sha256"`
		RunTimeout     string `json:"run_timeout"`
		MaxBodyBytes   int64  `json:"max_body_bytes"`
		ContextWorkers int    `json:"context_workers"`
	}{
		SchemaVersion:  ReadinessIdentitySchemaV1,
		Mode:           config.Mode,
		Source:         source,
		Application:    application,
		Runtime:        runtimeIdentity,
		Model:          modelIdentity,
		ServedModel:    servedModel,
		DataSHA256:     dataSHA,
		RunTimeout:     config.RunTimeout.String(),
		MaxBodyBytes:   config.MaxBodyBytes,
		ContextWorkers: config.Golden.ContextConcurrency,
	})
	if err != nil {
		return ReadinessIdentities{}, err
	}
	configDigest := sha256.Sum256(configPayload)
	return ReadinessIdentities{
		SchemaVersion:       ReadinessIdentitySchemaV1,
		Source:              source,
		Application:         application,
		Runtime:             runtimeIdentity,
		Model:               modelIdentity,
		ServedModel:         servedModel,
		ConfigurationSHA256: hex.EncodeToString(configDigest[:]),
		DataSHA256:          dataSHA,
	}, nil
}

func hashNamedPayloads(payloads map[string][]byte) (string, error) {
	if len(payloads) == 0 {
		return "", errors.New("readiness data identity requires at least one payload")
	}
	names := make([]string, 0, len(payloads))
	for name := range payloads {
		if strings.TrimSpace(name) == "" {
			return "", errors.New("readiness data identity contains an empty label")
		}
		names = append(names, name)
	}
	sort.Strings(names)
	digest := sha256.New()
	for _, name := range names {
		digest.Write([]byte(name))
		digest.Write([]byte{0})
		digest.Write(payloads[name])
		digest.Write([]byte{0})
	}
	return hex.EncodeToString(digest.Sum(nil)), nil
}
