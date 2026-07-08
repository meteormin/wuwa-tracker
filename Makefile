.PHONY: help setup webui-install webui-build webui-check webui-dev fmt fmt-check check clippy test ci build release run serve version release-dry-run bump-patch bump-minor bump-major release-tag clean distclean

APP := wuwa-tracker
WEBUI_DIR := crates/wuwa-tracker-webui
HOST ?= 127.0.0.1
PORT ?= 3000
WEBUI ?= 0
CARGO ?= cargo
TRUNK ?= trunk
WASM_TOOLCHAIN ?= 1.96.0
RUSTUP ?= $(shell if test -x /opt/homebrew/opt/rustup/bin/rustup; then echo /opt/homebrew/opt/rustup/bin/rustup; else command -v rustup 2>/dev/null || echo rustup; fi)
RUSTUP_BIN_DIR := $(dir $(RUSTUP))
WASM_ENV := PATH="$(RUSTUP_BIN_DIR):$$PATH" RUSTUP_TOOLCHAIN=$(WASM_TOOLCHAIN) NO_COLOR=false
SERVE_WEBUI := $(if $(filter 1 true yes,$(WEBUI)),--webui,)

help:
	@echo "Wuwa Tracker"
	@echo ""
	@echo "Development:"
	@echo "  make setup           Prepare the Rust WASM target"
	@echo "  make run             Build WebUI and run Tauri GUI"
	@echo "  make serve           Run Axum API server"
	@echo "  make webui-dev       Run Trunk dev server"
	@echo ""
	@echo "Checks:"
	@echo "  make fmt             Format Rust code"
	@echo "  make fmt-check       Check Rust formatting"
	@echo "  make check           cargo check + WebUI type check"
	@echo "  make clippy          cargo clippy"
	@echo "  make test            cargo test"
	@echo "  make ci              fmt-check + check + clippy + test"
	@echo ""
	@echo "Build:"
	@echo "  make build           Build WebUI and debug Rust workspace"
	@echo "  make release         Build WebUI and release Rust binary"
	@echo ""
	@echo "Versioning:"
	@echo "  make version         Print Cargo package version"
	@echo "  make release-dry-run Preview cargo-release changes"
	@echo "  make bump-patch      Bump patch version and create release commit"
	@echo "  make bump-minor      Bump minor version and create release commit"
	@echo "  make bump-major      Bump major version and create release commit"
	@echo "  make release-tag     Tag and push the version from synced main"
	@echo ""
	@echo "Options:"
	@echo "  make serve HOST=127.0.0.1 PORT=3000 WEBUI=1"
	@echo "  make setup WASM_TOOLCHAIN=1.96.0"

setup:
	$(RUSTUP) toolchain install $(WASM_TOOLCHAIN) --profile minimal --target wasm32-unknown-unknown
	@command -v $(TRUNK) >/dev/null || { echo "trunk is required: cargo install trunk --locked"; exit 1; }

webui-install:
	@command -v $(TRUNK) >/dev/null || { echo "trunk is required: run make setup after installing Trunk"; exit 1; }
	@$(RUSTUP) target list --installed --toolchain $(WASM_TOOLCHAIN) | grep -qx wasm32-unknown-unknown || { echo "wasm32-unknown-unknown is required: run make setup"; exit 1; }

webui-build: webui-install
	cd $(WEBUI_DIR) && $(WASM_ENV) $(TRUNK) build --release

webui-check: webui-install
	$(WASM_ENV) $(RUSTUP) run $(WASM_TOOLCHAIN) cargo check -p wuwa-tracker-webui --target wasm32-unknown-unknown

webui-dev: webui-install
	cd $(WEBUI_DIR) && $(WASM_ENV) $(TRUNK) serve

fmt:
	$(CARGO) fmt --all

fmt-check:
	$(CARGO) fmt --all -- --check

check: webui-check webui-build
	$(CARGO) check --workspace

clippy: webui-build
	$(CARGO) clippy --workspace --all-targets -- -D warnings

test: webui-build
	$(CARGO) test --workspace

ci: fmt-check check clippy test

build: webui-build
	$(CARGO) build --workspace

release: webui-build
	$(CARGO) build --release -p $(APP)

run: webui-build
	$(CARGO) run -p $(APP)

serve:
	$(CARGO) run -p $(APP) -- serve --host $(HOST) --port $(PORT) $(SERVE_WEBUI)

version:
	@$(CARGO) pkgid -p $(APP) | sed 's/.*#//; s/.*@//'

release-dry-run:
	$(CARGO) release patch --workspace --no-tag --dry-run

bump-patch:
	$(CARGO) release patch --workspace --no-tag --execute

bump-minor:
	$(CARGO) release minor --workspace --no-tag --execute

bump-major:
	$(CARGO) release major --workspace --no-tag --execute

release-tag:
	@git diff --quiet || { echo "Working tree has unstaged changes."; exit 1; }
	@git diff --cached --quiet || { echo "Index has staged changes."; exit 1; }
	@branch="$$(git branch --show-current)"; \
	if [ "$$branch" != "main" ]; then \
		echo "release-tag must run on main."; \
		exit 1; \
	fi
	@git fetch origin main --tags
	@local_head="$$(git rev-parse HEAD)"; \
	remote_head="$$(git rev-parse origin/main)"; \
	if [ "$$local_head" != "$$remote_head" ]; then \
		echo "main is not synced with origin/main. Push or merge the release commit before tagging."; \
		exit 1; \
	fi
	@version="$$($(CARGO) pkgid -p $(APP) | sed 's/.*#//; s/.*@//')"; \
	tag="v$$version"; \
	if git rev-parse -q --verify "refs/tags/$$tag" >/dev/null; then \
		echo "Tag $$tag already exists."; \
		exit 1; \
	fi; \
	git tag "$$tag"; \
	git push origin "$$tag"; \
	echo "Pushed $$tag."

clean:
	$(CARGO) clean
	rm -rf $(WEBUI_DIR)/dist

distclean: clean
