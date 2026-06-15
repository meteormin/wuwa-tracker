.PHONY: help setup webui-install webui-build webui-check webui-dev fmt fmt-check check clippy test ci build release run serve clean distclean

APP := wuwa-tracker
WEBUI_DIR := webui
HOST ?= 127.0.0.1
PORT ?= 3000
CARGO ?= cargo
YARN ?= yarn
YARN_INSTALL_FLAGS ?= --frozen-lockfile

help:
	@echo "Wuwa Tracker Rust v2"
	@echo ""
	@echo "Development:"
	@echo "  make setup           Install WebUI dependencies"
	@echo "  make run             Build WebUI and run Tauri GUI"
	@echo "  make serve           Build WebUI and run Axum WebUI server"
	@echo "  make webui-dev       Run Vite dev server"
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
	@echo "Options:"
	@echo "  make serve HOST=127.0.0.1 PORT=3000"
	@echo "  make setup YARN_INSTALL_FLAGS='--offline'"

setup webui-install:
	$(YARN) --cwd $(WEBUI_DIR) install $(YARN_INSTALL_FLAGS)

webui-build: webui-install
	$(YARN) --cwd $(WEBUI_DIR) run build

webui-check:
	$(YARN) --cwd $(WEBUI_DIR) run check

webui-dev:
	$(YARN) --cwd $(WEBUI_DIR) run dev

fmt:
	$(CARGO) fmt --all

fmt-check:
	$(CARGO) fmt --all -- --check

check: webui-check
	$(CARGO) check --workspace

clippy:
	$(CARGO) clippy --workspace --all-targets -- -D warnings

test:
	$(CARGO) test --workspace

ci: fmt-check check clippy test

build: webui-build
	$(CARGO) build --workspace

release: webui-build
	$(CARGO) build --release -p $(APP)

run: webui-build
	$(CARGO) run -p $(APP)

serve: webui-build
	$(CARGO) run -p $(APP) -- serve --host $(HOST) --port $(PORT) --webui-dist $(WEBUI_DIR)/dist

clean:
	$(CARGO) clean
	rm -rf $(WEBUI_DIR)/dist

distclean: clean
	rm -rf $(WEBUI_DIR)/node_modules .cache/yarn
