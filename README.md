# WUWA Tracker

Wuthering Waves(명조) 가챠 기록 URL을 수집하고, 배너별 기록을 분석해 pity, 5성 이력, 평균 소요, Luck Score를 보여주는 Rust 기반 로컬 트래커입니다.

## Project Structure

- `crates/wuwa-tracker-core`: 도메인 로직 부품, Kurogame API client, 로그 스캐너, 기록 병합, JSON store, 통계 계산, Askama HTML 리포트
- `crates/wuwa-tracker-types`: core, app, WebUI가 공유하는 도메인 모델과 직렬화 응답 계약
- `crates/wuwa-tracker-app`: 실행 파일, application service layer, Tauri GUI, Axum 기반 `serve` 서버, CLI subcommand
- `crates/wuwa-tracker-webui`: Leptos CSR 기반 Rust WASM UI. Tauri GUI와 Trunk 개발 서버에서 사용합니다.
- `locales`: Rust에 embed되는 locale JSON 리소스

기본 실행 단위는 Rust workspace입니다.

## Requirements

- Rust stable
- rustup의 `wasm32-unknown-unknown` target
- Trunk 0.21+

macOS Homebrew 환경에서는 다음과 같이 준비할 수 있습니다.

```bash
brew install rustup trunk
make setup
```

## Quick Start

```bash
make setup
make check
make run
```

`make run`은 WebUI를 빌드한 뒤 Tauri 데스크톱 앱을 실행합니다.

API 서버 모드를 테스트하려면:

```bash
make serve
```

기본 주소는 `http://127.0.0.1:3000`입니다. `serve`는 기본적으로 WebUI static asset을 제공하지 않고 `/api/*` route만 제공합니다.

포트를 바꾸려면:

```bash
make serve PORT=3999
```

WebUI만 개발할 때는 터미널을 둘로 나눠 실행합니다.

```bash
make serve
make webui-dev
```

Trunk 개발 서버는 API를 `http://localhost:3000`으로 호출합니다.

빌드 시 바이너리에 포함된 WebUI를 같은 서버에서 함께 제공하려면:

```bash
make build
cargo run -p wuwa-tracker -- serve --webui
```

Makefile로 실행할 때는:

```bash
make serve WEBUI=1
```

## Makefile

```bash
make help            # 사용 가능한 명령 출력
make setup           # Rust WASM toolchain 준비
make run             # WebUI 빌드 후 Tauri GUI 실행
make serve           # HTTP API 서버 실행, WEBUI=1이면 WebUI도 제공
make webui-dev       # Trunk 개발 서버 실행
make build           # WebUI + Rust workspace debug build
make release         # WebUI + optimized Rust binary build
make version         # Cargo package version 출력
make release-dry-run # cargo-release 변경 preview
make bump-patch      # patch version bump + release commit/tag
make bump-minor      # minor version bump + release commit/tag
make bump-major      # major version bump + release commit/tag
make check           # Cargo workspace + WASM target check
make clippy          # cargo clippy
make test            # cargo test
make ci              # fmt-check + check + clippy + test
make fmt             # cargo fmt
make clean           # Rust/WebUI build output 제거
make distclean       # Rust/WASM 산출물 제거
```

## Versioning

버전의 기준은 root `Cargo.toml`의 `[workspace.package] version`입니다. 각 crate는 `version.workspace = true`를 사용하고, CLI `version` 명령도 Cargo가 컴파일 시 주입하는 `CARGO_PKG_VERSION`을 출력합니다.

릴리즈 버전 bump는 `cargo-release`를 사용합니다.

```bash
cargo install cargo-release
make release-dry-run
make bump-patch
make bump-minor
make bump-major
```

`make release`는 현재 버전으로 로컬 release build만 수행하며 버전을 올리거나 tag를 만들지 않습니다. 실제 배포는 `vX.Y.Z` tag push로 시작됩니다.

```bash
make release-dry-run
make bump-patch
git push --follow-tags
```

GitHub release workflow는 tag push에서만 실행되며, `cargo pkgid -p wuwa-tracker`에서 읽은 버전과 tag 이름이 일치하는지 확인한 뒤 배포 산출물을 생성합니다.

## CLI

직접 실행 예시는 다음과 같습니다.

```bash
cargo run -p wuwa-tracker -- --help
cargo run -p wuwa-tracker -- version
cargo run -p wuwa-tracker -- scan --path "<game-root-or-log-path>"
cargo run -p wuwa-tracker -- report --url "<gacha-url>" --format html --output report --lang ko
cargo run -p wuwa-tracker -- report --file logs/history.json --format json --output out/history
cargo run -p wuwa-tracker -- run --path "<game-root-or-log-path>" --format html --output report
cargo run -p wuwa-tracker -- backup --output wuwa-tracker.backup.json
cargo run -p wuwa-tracker -- merge --file wuwa-tracker.backup.json
cargo run -p wuwa-tracker -- db stats
cargo run -p wuwa-tracker -- db players
cargo run -p wuwa-tracker -- db stats "<player-id>"
cargo run -p wuwa-tracker -- db records "<player-id>"
```

지원 리포트 포맷은 `html`, `json`, `csv`입니다.

전역 옵션으로 local store와 application log 경로를 바꿀 수 있습니다. 같은 값은 `WUWA_TRACKER_DB_PATH`, `WUWA_TRACKER_LOG_PATH` 환경 변수로도 지정할 수 있습니다.

```bash
cargo run -p wuwa-tracker -- --dbpath ./store.json --logpath ./wuwa-tracker.log serve
```

## API Spec

| Method | Path | 설명 |
| --- | --- | --- |
| `POST` | `/api/track` | `{ "url": "..." }`로 원격 가챠 기록 조회 후 저장 |
| `POST` | `/api/upload` | `FetchResult` JSON 업로드 후 저장 |
| `GET` | `/api/stats/{player_id}` | 저장 기록 기반 통계 조회 |
| `GET` | `/api/players` | 저장된 플레이어 목록 조회 |
| `GET` | `/api/config` | Luck Score threshold 설정 조회 |
| `GET` | `/api/i18n?lang={lang}` | UI 번역 조회 |
| `GET` | `/api/export/{player_id}?format={format}&lang={lang}` | HTML/JSON/CSV 리포트 다운로드 |
| `GET` | `/api/backup` | 로컬 store 백업 JSON 다운로드 |

## Store

기본 저장소는 `~/.wuwa-tracker/store.json`입니다.

필요한 경우 JSON export 또는 verbose log를 `report --file`이나 WebUI 업로드로 가져올 수 있습니다.

## Logging

기본 application log 경로는 `~/.wuwa-tracker/wuwa-tracker.log`입니다. App layer는 `tracing` event를 발생시킵니다. 일반 CLI 콘솔은 기본적으로 ERROR 이상, `serve` 콘솔과 파일 및 GUI runtime은 INFO 이상을 기록합니다. `RUST_LOG` 또는 `WUWA_TRACKER_LOG_LEVEL` 환경 변수를 지정하면 모든 subscriber의 runtime filter를 변경할 수 있으며, 둘 다 존재하면 `RUST_LOG`를 우선합니다.

```bash
RUST_LOG=debug cargo run -p wuwa-tracker -- db stats
RUST_LOG=wuwa_tracker_core=trace cargo run -p wuwa-tracker -- serve
WUWA_TRACKER_LOG_LEVEL=warn cargo run -p wuwa-tracker -- serve
```

파일 로그는 JSON Lines 형식이며, `serve` mode는 HTTP access log도 같은 파일에 기록합니다. Log file은 10 MiB를 넘기기 전에 rotation되며 최대 10개까지 보관합니다.

## Verification

```bash
make ci
make build
```

자세한 설계는 [DESIGN.md](DESIGN.md)를 참고하세요.
