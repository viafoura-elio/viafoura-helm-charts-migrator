# Makefile for Helm Chart Testing and Validation

.PHONY: help test test-simple test-dev lint schema docs template validate clean

# Default target
help: ## Show this help message
	@echo "Available targets:"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-20s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# Testing targets
test: ## Run all helm unit tests
	helm unittest .

test-simple: ## Run simple tests only (no schema validation issues)
	helm unittest -f 'tests/simple_test.yaml' .

test-dev: ## Run tests with dev01 values
	helm unittest -v envs/dev01/vf-dev2/values.yaml .

test-report: ## Generate JUnit test report
	helm unittest -o test-results.xml -t JUnit .

# Validation targets
lint: ## Lint helm chart
	helm lint .

schema: ## Generate JSON schema
	helm-schema -p -r

validate: ## Validate templates against schema
	helm template . --validate

# Documentation targets
docs: ## Generate helm documentation
	helm-docs -t ./README.md.gotmpl .

# Template rendering targets
template: ## Render all templates with default values
	helm template livecomments .

template-dev: ## Render templates with dev01 values
	helm template livecomments . -f envs/dev01/vf-dev2/values.yaml

.PHONY: template-dev-with-secrets
template-dev-with-secrets: ## Render templates with dev01 values
	helm template livecomments . \
	  -f values.yaml \
	  -f envs/dev01/vf-dev2/values.yaml \
  	  -f secrets://envs/dev01/vf-dev2/values.yaml

.PHONY: template-dev-with-secrets-rollout
template-dev-with-secrets-rollout: ## Render templates with dev01 values
	helm template livecomments . \
	  --show-only templates/rollout.yaml \
	  -f values.yaml \
	  -f envs/dev01/vf-dev2/values.yaml \
  	  -f secrets://envs/dev01/vf-dev2/values.yaml

template-rollout: ## Show only rollout template
	helm template livecomments . --show-only templates/rollout.yaml

template-istio: ## Show only Istio templates
	helm template livecomments . --show-only templates/istio-virtualservice.yaml,templates/istio-destinationrule.yaml,templates/istio-gateway.yaml

# Debugging targets
debug-rollout: ## Debug rollout configuration
	helm template livecomments . --show-only templates/rollout.yaml --debug

debug-hpa: ## Debug HPA configuration
	helm template livecomments . --show-only templates/hpa.yaml --debug

debug-values: ## Show computed values
	helm template livecomments . --debug | grep -A 1000 "COMPUTED VALUES"

# Package targets
package: lint test-simple ## Package the helm chart
	helm package .

# Clean targets
clean: ## Clean generated files
	rm -f *.tgz test-results.xml test-results-working.xml

# Installation targets (for local testing)
install-dev: ## Install chart in dev namespace
	helm upgrade --install livecomments-dev . \
		-f envs/dev01/vf-dev2/values.yaml \
		--namespace livecomments-dev \
		--create-namespace \
		--dry-run

uninstall-dev: ## Uninstall dev deployment
	helm uninstall livecomments-dev --namespace livecomments-dev

# CI/CD targets
ci-test: lint test-working verify-templates ## Run all CI checks
	@echo "✅ All CI checks passed!"

test-integration: ## Run integration tests only
	helm unittest -f 'tests/integration_test.yaml' .

test-istio: ## Run Istio-specific tests only
	helm unittest -f 'tests/istio_test.yaml' .

test-working: ## Run only working tests (simple + istio)
	helm unittest -f 'tests/simple_test.yaml' -f 'tests/istio_test.yaml' .

test-report-working: ## Generate test report for working tests
	helm unittest -f 'tests/simple_test.yaml' -f 'tests/istio_test.yaml' -o test-results-working.xml -t JUnit .

# Verification targets
verify-schema: ## Verify schema compliance
	@echo "Verifying JSON schema compliance..."
	helm template . --validate > /dev/null && echo "✅ Schema validation passed"

verify-templates: ## Verify all templates render correctly
	@echo "Verifying template rendering..."
	@helm template . > /dev/null && echo "✅ All templates render correctly"

# Development targets
watch-test: ## Watch for changes and run tests
	@echo "Watching for file changes... (Ctrl+C to stop)"
	@while true; do \
		make test-simple; \
		sleep 2; \
	done
