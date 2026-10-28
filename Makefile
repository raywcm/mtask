GO ?= go
PYTHON ?= python3

.PHONY: build test check integration acceptance notices snapshot clean
build:
	@mkdir -p bin
	@tmp=$$(mktemp ./bin/.mtask-build.XXXXXX); trap 'rm -f "$$tmp"' EXIT; $(GO) build -trimpath -o "$$tmp" ./cmd/mtask && mv -f "$$tmp" bin/mtask

test:
	$(GO) test -race ./...

check:
	$(GO) vet ./...
	$(GO) test -race ./...
	$(PYTHON) scripts/third_party_notices.py --check --go $(GO)

integration: build
	$(PYTHON) scripts/integration.py bin/mtask

acceptance: build
	$(PYTHON) scripts/acceptance.py --binary bin/mtask

notices:
	$(PYTHON) scripts/third_party_notices.py --go $(GO)

snapshot:
	goreleaser release --snapshot --clean --skip=publish
	python3 scripts/apt_repo.py dist dist/apt
	python3 scripts/check_packages.py dist

clean:
	rm -rf bin dist coverage.out
