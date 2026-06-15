# WUWA Tracker

Wuthering Waves(명조) 가챠 기록 URL을 수집하고, 배너별 기록을 분석해 pity, 5성 이력, 평균 소요, Luck Score를 보여주는 Rust 기반 로컬 트래커입니다.

## Project Structure

- `crates/wuwa-tracker-core`: 도메인 로직, Kurogame API client, 로그 스캐너, 기록 병합, JSON store, 통계 계산, Askama HTML 리포트
- `crates/wuwa-tracker-app`: 실행 파일, Tauri GUI, embedded WebUI를 제공하는 Axum 기반 `serve` 서버, CLI subcommand
- `webui`: Svelte + Vite UI. 삭제하지 않고 GUI와 서버 모드에서 함께 사용합니다.
- `locales`: Rust에 embed되는 locale JSON 리소스

Go 구현은 제거되었고, 기본 실행 단위는 Rust workspace입니다.

## Requirements

- Rust stable
- Node.js
- Yarn classic 또는 Corepack이 제공하는 Yarn

Yarn 실행 파일이 `yarn`이 아니라면 Makefile에서 `YARN`을 지정할 수 있습니다.

```bash
make check YARN="corepack yarn"
```

네트워크가 없는 로컬 환경에서 이미 의존성이 설치되어 있다면 Yarn install flag를 바꿀 수 있습니다.

```bash
make setup YARN_INSTALL_FLAGS="--offline"
make build YARN_INSTALL_FLAGS="--offline"
```

## Quick Start

```bash
make setup
make check
make run
```

`make run`은 WebUI를 빌드한 뒤 Tauri 데스크톱 앱을 실행합니다.

서버 모드를 테스트하려면:

```bash
make serve
```

기본 주소는 `http://127.0.0.1:3000`입니다. `serve`의 WebUI asset은 빌드 시 바이너리에 embed되므로, 빌드된 실행 파일은 별도 `webui/dist` 디렉터리 없이 실행할 수 있습니다.

포트를 바꾸려면:

```bash
make serve PORT=3999
```

WebUI만 개발할 때는 터미널을 둘로 나눠 실행합니다.

```bash
make serve
make webui-dev
```

Vite 개발 서버는 API를 `http://localhost:3000`으로 호출합니다.

## Makefile

```bash
make help            # 사용 가능한 명령 출력
make setup           # webui 의존성 설치
make run             # WebUI 빌드 후 Tauri GUI 실행
make serve           # WebUI 빌드 후 HTTP WebUI 서버 실행
make webui-dev       # Vite 개발 서버 실행
make build           # WebUI + Rust workspace debug build
make release         # WebUI + optimized Rust binary build
make version         # Cargo package version 출력
make release-dry-run # cargo-release 변경 preview
make bump-patch      # patch version bump + release commit/tag
make bump-minor      # minor version bump + release commit/tag
make bump-major      # major version bump + release commit/tag
make check           # cargo check + Svelte/TypeScript check
make clippy          # cargo clippy
make test            # cargo test
make ci              # fmt-check + check + clippy + test
make fmt             # cargo fmt
make clean           # Rust/WebUI build output 제거
make distclean       # dependency cache와 node_modules까지 제거
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
cargo run -p wuwa-tracker -- db players
```

지원 리포트 포맷은 `html`, `json`, `csv`입니다.

## WebUI API

| Method | Path | 설명 |
| --- | --- | --- |
| `POST` | `/api/track` | `{ "url": "..." }`로 원격 가챠 기록 조회 후 저장 |
| `POST` | `/api/upload` | `FetchResult` JSON 업로드 후 저장 |
| `GET` | `/api/stats/{player_id}` | 저장 기록 기반 통계 조회 |
| `GET` | `/api/players` | 저장된 플레이어 목록 조회 |
| `GET` | `/api/config` | Luck Score threshold 설정 조회 |
| `GET` | `/api/i18n?lang={lang}` | UI 번역 조회 |
| `GET` | `/api/export/{player_id}?format={format}&lang={lang}` | HTML/JSON/CSV 리포트 다운로드 |

## Store

Rust v2는 기본적으로 `~/.wuwa-tracker/v2-store.json`을 사용합니다.

기존 Go/BadgerDB 데이터와의 자동 마이그레이션은 아직 제공하지 않습니다. 필요한 경우 기존 JSON export 또는 verbose log를 `report --file`이나 WebUI 업로드로 가져오면 됩니다.

## Verification

```bash
cargo check --workspace
cargo test --workspace
yarn --cwd webui run check
```

자세한 설계는 [DESIGN.md](DESIGN.md)를 참고하세요.
