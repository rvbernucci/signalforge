PROFILE ?= radeon-local
BACKEND ?= auto
ACCEPT_GEMMA_LICENSE ?= no
RADEON_NONINTERACTIVE ?= 0
MANIFEST ?=

.PHONY: verify fixture-up fixture-down radeon-bootstrap radeon-preflight radeon-up \
	radeon-local-up championship-up radeon-status radeon-logs radeon-observe \
	radeon-down radeon-clean radeon-reset stack-down mission-control-up \
	mission-control-down stack-status stack-logs evidence-export radeon-exporter \
	observability-check container-check

verify:
	scripts/verify.sh

fixture-up:
	SIGNALFORGE_PROFILE=fixture SIGNALFORGE_EXECUTION_BACKEND="$(BACKEND)" \
		SIGNALFORGE_APPLIANCE_MANIFEST="$(MANIFEST)" \
		scripts/radeon_up.sh

fixture-down:
	SIGNALFORGE_PROFILE=fixture SIGNALFORGE_EXECUTION_BACKEND="$(BACKEND)" \
		scripts/radeon_down.sh

radeon-bootstrap:
	SIGNALFORGE_ACCEPT_GEMMA_LICENSE="$(ACCEPT_GEMMA_LICENSE)" \
		SIGNALFORGE_APPLIANCE_MANIFEST="$(MANIFEST)" \
		python3 scripts/radeon_bootstrap.py \
		--profile "$(PROFILE)" \
		--backend "$(BACKEND)" \
		$(if $(strip $(MANIFEST)),--manifest "$(MANIFEST)",) \
		$(if $(filter yes,$(ACCEPT_GEMMA_LICENSE)),--accept-gemma-license,) \
		$(if $(filter 1,$(RADEON_NONINTERACTIVE)),--noninteractive,)

radeon-preflight:
	SIGNALFORGE_PROFILE="$(PROFILE)" SIGNALFORGE_EXECUTION_BACKEND="$(BACKEND)" \
		SIGNALFORGE_APPLIANCE_MANIFEST="$(MANIFEST)" \
		scripts/radeon_preflight.sh

radeon-up:
	SIGNALFORGE_PROFILE="$(PROFILE)" SIGNALFORGE_EXECUTION_BACKEND="$(BACKEND)" \
		SIGNALFORGE_APPLIANCE_MANIFEST="$(MANIFEST)" \
		scripts/radeon_up.sh

radeon-local-up:
	SIGNALFORGE_PROFILE=radeon-local SIGNALFORGE_EXECUTION_BACKEND="$(BACKEND)" \
		SIGNALFORGE_APPLIANCE_MANIFEST="$(MANIFEST)" \
		scripts/radeon_up.sh

championship-up:
	SIGNALFORGE_PROFILE=championship SIGNALFORGE_EXECUTION_BACKEND="$(BACKEND)" \
		SIGNALFORGE_APPLIANCE_MANIFEST="$(MANIFEST)" \
		scripts/radeon_up.sh

radeon-status:
	SIGNALFORGE_APPLIANCE_MANIFEST="$(MANIFEST)" \
		python3 scripts/radeon_status.py --profile "$(PROFILE)" --backend "$(BACKEND)"

radeon-logs:
	SIGNALFORGE_PROFILE="$(PROFILE)" SIGNALFORGE_EXECUTION_BACKEND="$(BACKEND)" \
		scripts/radeon_logs.sh

radeon-observe:
	SIGNALFORGE_PROFILE="$(PROFILE)" SIGNALFORGE_EXECUTION_BACKEND="$(BACKEND)" \
		SIGNALFORGE_APPLIANCE_MANIFEST="$(MANIFEST)" \
		SIGNALFORGE_OBSERVABILITY=1 \
		SIGNALFORGE_OTEL_ENABLED=true scripts/radeon_up.sh

radeon-down:
	SIGNALFORGE_EXECUTION_BACKEND="$(BACKEND)" scripts/radeon_down.sh

radeon-clean: radeon-down
	python3 scripts/radeon_reset.py --mode clean --confirm "$(CONFIRM)"

radeon-reset: radeon-down
	python3 scripts/radeon_reset.py --mode reset --confirm "$(CONFIRM)"

stack-down: radeon-down

mission-control-up:
	scripts/prepare_container_secrets.sh
	mkdir -p .signalforge/radeon
	SIGNALFORGE_PROFILE=fixture SIGNALFORGE_OBSERVABILITY=1 \
		scripts/radeon_compose.sh current up --detach --no-build

mission-control-down:
	SIGNALFORGE_PROFILE=fixture SIGNALFORGE_OBSERVABILITY=1 \
		scripts/radeon_compose.sh current down

stack-status: radeon-status

stack-logs: radeon-logs

evidence-export:
	python3 scripts/export_mission_control_evidence.py \
		--audit-dir .signalforge/intelligence-audit \
		--output output/mission-control-evidence.tar.gz

radeon-exporter:
	RADEON_EXPORTER_HOST=127.0.0.1 python3 deploy/observability/radeon-exporter/exporter.py

observability-check:
	python3 scripts/validate_observability.py

container-check:
	scripts/verify_container_fixture.sh
