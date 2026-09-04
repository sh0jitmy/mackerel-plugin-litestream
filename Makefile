# Makefile for Go Development & Custom Skills Management

.PHONY: help check install self-eval generate test fmt lint tidy vulncheck build release-check release-snapshot license-check license-add migration-diff clean openapi-lint publish-pr ai-pr

help:
	@echo "Available commands:"
	@echo "  Go Development:"
	@echo "    openapi-lint     Validate OpenAPI spec with Spectral"
	@echo "    generate         Generate OpenAPI and ent entity code"
	@echo "    fmt              Format Go source files"
	@echo "    lint             Run golangci-lint static analysis"
	@echo "    tidy             Run go mod tidy"
	@echo "    vulncheck        Run govulncheck vulnerability scanner"
	@echo "    test             Run Go tests with race detector and coverage"
	@echo "    build            Build binary to bin/app"
	@echo "    release-check    Validate GoReleaser configuration"
	@echo "    release-snapshot Run GoReleaser snapshot build"
	@echo "    license-check    Verify license & author headers in Go files"
	@echo "    license-add      Automatically add license headers to Go files"
	@echo "    migration-diff   Generate DB migration SQL file with Atlas (requires Atlas CLI)"
	@echo "                     Usage: make migration-diff name=migration_name"
	@echo "    publish-pr       Verify formatting/lints/tests, push to origin, and create GitHub PR"
	@echo "    ai-pr            Trigger AI agent to analyze commits/diffs and create a draft GitHub PR in Japanese"
	@echo "  Custom Skills Management:"
	@echo "    check            Validate custom skill frontmatter and syntax"
	@echo "    install          Install custom skills globally to ~/.claude/skills/"
	@echo "    self-eval        Run requirements self-evaluation and update checklist"
	@echo "  General:"
	@echo "    clean            Clean up build artifacts and temporary files"

# --- Go Development ---

openapi-lint:
	@echo "==> Running Spectral lint on OpenAPI spec..."
	@if command -v spectral >/dev/null 2>&1; then \
		NODE_OPTIONS="--no-deprecation" spectral lint api/openapi.yaml; \
	elif command -v npx >/dev/null 2>&1; then \
		NODE_OPTIONS="--no-deprecation" npx -y @stoplight/spectral-cli lint api/openapi.yaml; \
	else \
		echo "Spectral CLI is not installed and npx is not available. Please install it."; \
		exit 1; \
	fi

generate: openapi-lint
	@echo "==> Generating code from schema..."
	@go generate ./...

fmt: generate
	@echo "==> Formatting Go source files..."
	@go fmt ./...
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run --fix ./...; \
	fi

lint: generate
	@echo "==> Running golangci-lint..."
	@golangci-lint run ./...

tidy:
	@echo "==> Tidying Go modules..."
	@go mod tidy

vulncheck:
	@echo "==> Running govulncheck..."
	@go run golang.org/x/vuln/cmd/govulncheck@latest ./...

test: generate
	@bash scripts/check_coverage.sh

build: generate
	@echo "==> Building binary..."
	@mkdir -p bin
	@go build -v -o bin/app ./cmd/app

release-check:
	@echo "==> Validating GoReleaser configuration..."
	@goreleaser check

release-snapshot:
	@echo "==> Building GoReleaser snapshot..."
	@goreleaser release --snapshot --clean

license-check:
	@echo "==> Checking Go source files license headers..."
	@python3 scripts/check_license.py --check

license-add:
	@echo "==> Adding license headers to Go source files..."
	@python3 scripts/check_license.py --add

migration-diff:
	@if [ -z "$(name)" ]; then \
		echo "Error: name is required. Usage: make migration-diff name=migration_name"; \
		exit 1; \
	fi
	@if ! command -v atlas >/dev/null 2>&1; then \
		echo "Atlas CLI is not installed. Please install it from: https://atlasgo.io/"; \
		exit 1; \
	fi
	@echo "==> Generating DB migration DDL with Atlas..."
	@mkdir -p ent/migrate/migrations
	@atlas migrate diff $(name) \
		--dir "file://ent/migrate/migrations" \
		--to "ent://ent/schema" \
		--dev-url "sqlite://dev?mode=memory"

publish-pr:
	@bash scripts/publish_pr.sh

ai-pr:
	@claude "github-pr-creator スキルを使用して、現在のブランチの変更とコミットログを分析し、pull_request_template.md に従って日本語のプルリクエストをドラフト（下書き）で作成してください。"

# --- Custom Skills Management ---

check:
	@echo "==> Validating skill files format..."
	@python3 scripts/check_skills.py

install:
	@echo "==> Installing custom skills globally to ~/.claude/skills/..."
	@mkdir -p ~/.claude/skills/
	@cp -R .claude/skills/* ~/.claude/skills/
	@echo "Skills successfully installed!"

self-eval:
	@echo "==> Running self-evaluation..."
	@python3 scripts/self_eval.py

# --- General ---

clean:
	@echo "==> Cleaning up build artifacts..."
	@rm -rf bin/ dist/ ent/migrate/migrations/
	# Keep generated code unless explicit reset is wanted
	@go clean -testcache
