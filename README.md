# wuwa-tracker

Wuthering Waves(명조) 튜닝 기록 URL을 수집하고, 배너별 가챠 기록을 분석해 천장 스택, 5성 획득 이력, 평균 소요, 운 점수(Luck Score)를 보여주는 로컬 트래커입니다.

프로젝트는 단일 실행 파일을 제공합니다.

- `wuwa-tracker`: 기본 실행 시 Svelte WebUI와 Go Fiber API를 포함한 로컬 웹 서버를 실행하며, 서브커맨드로 로그 스캔, 백업 병합, DB 관리, 리포트 생성을 제공합니다.

## 주요 기능

- 게임 로그에서 가챠 기록 URL 추출
- Kurogame 가챠 기록 API 조회
- HTML, JSON, CSV 리포트 생성
- 로컬 JSON 로그 파일 기반 오프라인 리포트 생성
- BadgerDB 기반 플레이어별 기록 저장 및 증분 병합
- Svelte WebUI에서 URL 동기화, JSON 업로드, 저장 플레이어 조회, 리포트 다운로드
- 임베디드 설정과 로케일 fallback을 통한 단일 바이너리 배포

## 요구 사항

- Go 1.26.x
- Node.js 및 Yarn classic
- `golangci-lint`는 `make lint` 실행 시 필요

Go 빌드 캐시, Go 모듈 캐시, Yarn 캐시는 기본적으로 프로젝트 내부 `.cache/`를 사용합니다.

## 빌드와 검증

```bash
make build          # WebUI 빌드 후 bin/wuwa-tracker 생성
make build COMMIT_HASH=$(git rev-parse --short HEAD)
make test           # Go 테스트
make lint           # golangci-lint 실행
make audit          # go mod verify, go vet, govulncheck 실행
make clean          # bin, Go/Yarn 캐시, webui/node_modules, webui/dist 제거
```

빌드 산출물에는 기본적으로 `YYYYMMDD` 형식의 빌드 태그가 주입됩니다. `COMMIT_HASH`를 지정하면 `YYYYMMDD-<hash>` 형식으로 주입되며, `./bin/wuwa-tracker version` 또는 서버 실행 시 출력되는 배너에서 확인할 수 있습니다.

WebUI만 개발할 때는 `webui` 디렉터리에서 실행합니다.

```bash
cd webui
yarn install
yarn dev
yarn check
```

Vite 개발 서버는 API 호출을 `http://localhost:3000`의 Go 서버로 보냅니다.

## 사용법

기본 명령은 서버 실행입니다.

```bash
./bin/wuwa-tracker
./bin/wuwa-tracker -host 0.0.0.0 -port 3000
./bin/wuwa-tracker serve -host 0.0.0.0 -port 3000
```

그 외 기능은 서브커맨드 기반입니다.

```bash
./bin/wuwa-tracker <command> [arguments]
```

### 로그 URL 스캔

```bash
./bin/wuwa-tracker scan -path "<게임 설치 또는 로그 경로>"
./bin/wuwa-tracker scan -path "<게임 설치 또는 로그 경로>" -clipboard
```

`scan`은 `Client.log`, `debug.log` 등 지원 경로에서 마지막 가챠 기록 URL을 찾아 표준 출력으로 반환합니다. `-clipboard`를 사용하면 OS 기본 클립보드 도구(`pbcopy`, `clip`, `xclip`, `wl-copy`)로 복사합니다.

### 온라인 리포트 생성

```bash
./bin/wuwa-tracker report -url "가챠URL" -format html -o report -lang ko
./bin/wuwa-tracker report -url "가챠URL" -format json -o out/history
./bin/wuwa-tracker report -url "가챠URL" -format csv -o out/history
```

지원 포맷은 `html`, `json`, `csv`입니다. 조회 결과는 기본 `~/.wuwa-tracker/badger` DB에 병합 저장되며, `-dbpath`로 저장 경로를 바꿀 수 있습니다. `-v`를 추가하면 조회 결과를 `logs/<playerId>-<timestamp>.json`으로도 저장합니다.

### 오프라인 리포트 생성

```bash
./bin/wuwa-tracker report -f logs/history.json -format html -o report -lang en
```

오프라인 모드는 `FetchResult` 형식과 legacy `map[string][]Record` 형식 JSON을 모두 읽을 수 있으며, 읽은 기록을 DB에 병합 저장한 뒤 리포트를 생성합니다.

### 전체 흐름 실행

```bash
./bin/wuwa-tracker run -path "<게임 설치 또는 로그 경로>" -format html -o report -lang ko
./bin/wuwa-tracker run -url "가챠URL" -format html -o report
```

`run`은 URL이 없으면 경로에서 URL을 스캔한 뒤, 기록 조회, DB 병합 저장, 리포트 생성을 이어서 수행합니다.

### DB 백업과 병합

```bash
./bin/wuwa-tracker backup -o wuwa-tracker.backup
./bin/wuwa-tracker merge -f wuwa-tracker.backup
```

`backup`은 현재 BadgerDB를 단일 백업 파일로 내보냅니다. `merge`는 백업 파일을 임시 DB에 복원한 뒤 현재 DB의 기존 기록과 배너별로 병합합니다. 기본 DB 경로는 `~/.wuwa-tracker/badger`이며, 두 명령 모두 `-dbpath`로 저장 경로를 바꿀 수 있습니다.

## WebUI 서버 사용법

```bash
./bin/wuwa-tracker
./bin/wuwa-tracker -host 127.0.0.1 -port 9090 -dbpath "$HOME/.wuwa-tracker/badger"
./bin/wuwa-tracker serve -host 127.0.0.1 -port 9090 -dbpath "$HOME/.wuwa-tracker/badger"
WUWA_TRACKER_HOST=127.0.0.1 WUWA_TRACKER_PORT=9090 WUWA_TRACKER_DB_PATH="$HOME/.wuwa-tracker/badger" ./bin/wuwa-tracker
```

환경 변수 `WUWA_TRACKER_HOST`, `WUWA_TRACKER_PORT`, `WUWA_TRACKER_DB_PATH`도 기본값으로 사용할 수 있습니다. CLI 플래그 `-host`, `-port`, `-dbpath`를 함께 지정하면 플래그 값이 우선합니다. 서버는 기본적으로 `127.0.0.1:3000`에서만 수신하며, 개발용 CORS 허용 origin은 `WUWA_TRACKER_CORS_ORIGINS`에 쉼표로 구분해 지정할 수 있습니다.

WebUI에서 가능한 작업:

- 가챠 URL 입력 후 원격 기록 동기화
- CLI verbose 로그 등 JSON 파일 업로드
- 저장된 플레이어 목록 조회 및 전환
- HTML, JSON, CSV 리포트 다운로드
- 서버 설정의 운 점수 임계값 기반 상태 표시
- 서버 embed i18n 리소스 기반 UI/리포트 문구 표시

## API 개요

| Method | Path | 설명 |
| --- | --- | --- |
| `POST` | `/api/track` | `{ "url": "..." }`로 모든 배너 기록을 조회하고 DB에 병합 저장 |
| `POST` | `/api/upload` | `FetchResult` JSON을 업로드하여 배너별 기록 저장 |
| `GET` | `/api/stats/:playerId` | 저장된 기록 기반 통계 조회 |
| `GET` | `/api/players` | 저장된 플레이어 ID 목록 조회 |
| `GET` | `/api/config` | 프론트 표시용 `luckScoreThresholds` 조회 |
| `GET` | `/api/i18n?lang=ko|en` | WebUI와 HTML 리포트에서 공유하는 UI 번역 조회 |
| `GET` | `/api/export/:playerId?format=html|json|csv&lang=ko|en` | 저장된 기록의 리포트 다운로드 |

## 프로젝트 구조

```text
cmd/
  cli/       CLI 진입점과 scan/report/run 서브커맨드
  server/    Fiber 서버 진입점
config/      기본 설정과 환경 변수 override
internal/
  db/        BadgerDB 저장소와 기록 병합
  handler/   HTTP API 핸들러와 에러 응답
  reporter/  HTML/JSON/CSV exporter
  scanner/   로그 파일 URL 스캐너와 시스템 로케일 감지
  tracker/   가챠 API 클라이언트, 로케일 fallback, 통계 계산
  types/     공통 데이터 모델
locales/     임베디드 가챠 배너명 로케일과 UI 번역 리소스
templates/   HTML 리포트 템플릿
webui/       Svelte + Vite 프론트엔드와 임베디드 정적 파일
```

자세한 설계와 개발 참고 사항은 [DESIGN.md](DESIGN.md)를 참고하세요.
