GO ?= go
VERSION ?= 0.1.0-dev
BIN ?= bin/argus-asm
DEMO_DB ?= build/demo/argus-asm-demo.db
MODULE := github.com/NexzaDev/argus-asm
LDFLAGS := -X $(MODULE)/internal/buildinfo.Version=$(VERSION)

.PHONY: fmt fmt-check vet test race fuzz-smoke staticcheck govulncheck build build-nogui demo dist check clean

fmt:
	gofmt -w cmd internal migrations

fmt-check:
	@unformatted="$$(gofmt -l cmd internal migrations)" || exit 1; \
	if [ -n "$$unformatted" ]; then echo "gofmt needed:"; echo "$$unformatted"; exit 1; fi

vet:
	$(GO) vet -tags ci ./...

test:
	$(GO) test -count=1 -tags ci ./...

test-nogui:
	$(GO) test -count=1 -tags nogui ./...

race:
	$(GO) test -race -count=1 -tags ci ./...

fuzz-smoke:
	$(GO) test -run=^$$ -fuzz='^FuzzParseFindings$$' -fuzztime=10s ./internal/importer/
	$(GO) test -run=^$$ -fuzz='^FuzzParseFindingsJSON$$' -fuzztime=10s ./internal/importer/
	$(GO) test -run=^$$ -fuzz='^FuzzParseNmapXML$$' -fuzztime=10s ./internal/importer/
	$(GO) test -run=^$$ -fuzz='^FuzzParseFindingsSARIF$$' -fuzztime=10s ./internal/importer/

staticcheck:
	$(GO) run honnef.co/go/tools/cmd/staticcheck@2026.2.2 ./...

govulncheck:
	$(GO) run golang.org/x/vuln/cmd/govulncheck@v1.9.0 ./...

build:
	$(GO) build -trimpath -ldflags "$(LDFLAGS)" -o $(BIN) ./cmd/argus-asm

build-nogui:
	$(GO) build -trimpath -tags nogui -ldflags "$(LDFLAGS)" -o $(BIN) ./cmd/argus-asm

demo: build
	@echo "DEMO: all data below is synthetic (RFC 2606 names, RFC 5737 addresses) from testdata/demo/"
	rm -f $(DEMO_DB) build/demo/outbox.jsonl
	ARGUS_ASM_DB=$(DEMO_DB) ./$(BIN) init
	ARGUS_ASM_DB=$(DEMO_DB) ./$(BIN) scope add example.test
	ARGUS_ASM_DB=$(DEMO_DB) ./$(BIN) scope add 192.0.2.0/24
	ARGUS_ASM_DB=$(DEMO_DB) ./$(BIN) asset add www.example.test
	ARGUS_ASM_DB=$(DEMO_DB) ./$(BIN) asset add api.example.test
	ARGUS_ASM_DB=$(DEMO_DB) ./$(BIN) asset add 192.0.2.10
	ARGUS_ASM_DB=$(DEMO_DB) ./$(BIN) service import-nmap testdata/demo/nmap.xml
	ARGUS_ASM_DB=$(DEMO_DB) ./$(BIN) cert import www.example.test testdata/demo/certificate.pem
	ARGUS_ASM_DB=$(DEMO_DB) ./$(BIN) findings import --source demo-scanner testdata/demo/findings.csv
	ARGUS_ASM_DB=$(DEMO_DB) ./$(BIN) findings import --format json --source demo-json-feed testdata/demo/findings.json
	ARGUS_ASM_DB=$(DEMO_DB) ./$(BIN) findings import --format sarif --source demo-sarif-analyzer testdata/demo/findings.sarif
	ARGUS_ASM_DB=$(DEMO_DB) ./$(BIN) findings list --explain
	ARGUS_ASM_DB=$(DEMO_DB) ./$(BIN) notify refresh
	ARGUS_ASM_DB=$(DEMO_DB) ./$(BIN) notify deliver --to build/demo/outbox.jsonl
	ARGUS_ASM_DB=$(DEMO_DB) ./$(BIN) audit verify

dist:
	@name="argus-asm-$(VERSION)-$$($(GO) env GOOS)-$$($(GO) env GOARCH)"; \
	rm -rf "dist/$$name" "dist/$$name.tar.gz" "dist/$$name.tar.gz.sha256"; \
	mkdir -p "dist/$$name" && \
	$(GO) build -trimpath -ldflags "$(LDFLAGS)" -o "dist/$$name/argus-asm" ./cmd/argus-asm && \
	"dist/$$name/argus-asm" version && \
	cp LICENSE README.md THIRD_PARTY_NOTICES.md "dist/$$name/" && \
	tar -C dist -czf "dist/$$name.tar.gz" "$$name" && \
	rm -rf "dist/$$name" && \
	cd dist && { sha256sum "$$name.tar.gz" 2>/dev/null || shasum -a 256 "$$name.tar.gz"; } > "$$name.tar.gz.sha256" && \
	cat "$$name.tar.gz.sha256"

check: fmt-check vet test build-nogui

clean:
	rm -rf bin build dist
