# wuwa-tracker

명조: 워더링 웨이브(Wuthering Waves)의 튜닝(가챠) 기록을 수집하고 분석하여 누적 스택, 천장(Pity) 계산, 획득한 5성 캐릭터/무기 히스토리를 시각화하는 도구 패키지입니다. 

실행 파일 내에 내장된 Svelte WebUI 웹 서버와 CLI 수집 도구를 제공합니다.

---

## 주요 기능 및 구성

본 프로젝트는 외부 런타임 의존성이 없는 순수 Go 단일 바이너리(CGO 미사용)로 빌드하여 배포 및 실행할 수 있습니다.

### 1. WebUI 및 API 서버 (`wuwa-tracker-server`)
* **임베디드 프론트엔드**: Svelte 기반으로 개발된 대시보드 정적 리소스를 Go 바이너리 내부에 컴파일 시점(`go:embed`)에 내장하여 별도의 Node.js 설치 없이 실행 가능합니다.
* **로컬 API 서버**: Go Fiber 프레임워크 기반의 REST API 서버를 내장하고 있습니다.
* **오프라인 데이터 처리**: 기저장된 로컬 가챠 JSON 파일을 웹 UI에 업로드하여 분석을 수행하고 로컬 데이터베이스(BadgerDB)에 저장할 수 있습니다.
* **동작 제어**: 포트(`-port`)와 데이터베이스 저장 경로(`-dbpath`)를 CLI 플래그로 변경할 수 있습니다.

### 2. CLI 분석 도구 (`wuwa-tracker`)
* **게임 로그 스캔**: 명조 설치 경로(`-path`)를 지정하면 게임 로그 파일(`Client.log` / `debug.log`)에서 튜닝 기록 URL을 자동으로 탐색하여 분석합니다.
* **데이터 내보내기**: 수집된 가챠 상세 정보를 CSV 및 JSON 포맷의 1차원 데이터로 평탄화하여 출력합니다.
* **운 점수 (Luck Score) 계산**: 한정 캐릭터 배너의 픽업 사이클(상시 5성 픽뚫 확률 포함)을 고려하여 스택을 누적 산정하고 설정된 기댓값과 비교한 점수를 산출합니다.

### 3. 오프라인 리포트 생성 (`wuwa-tracker report -f`)
* CLI의 `report` 서브커맨드에 `-f` 플래그로 로컬 JSON 파일을 지정하면, 외부 API 요청 없이 오프라인으로 HTML/CSV/JSON 리포트를 생성할 수 있습니다.

---

## 개발 환경 및 빌드 방법

### 개발 환경 (Prerequisites)
* **Go**: 1.20 버전 이상
* **Node.js & Yarn** (웹 UI 빌드 시 필요): v16 이상

### 빌드 및 실행 명령어

제공되는 [Makefile](Makefile)을 사용하여 필요한 타겟을 빌드할 수 있습니다.

```bash
# 1. 프론트엔드 컴파일 및 Go 임베딩 빌드 일괄 실행
# Svelte 소스 빌드 -> dist 폴더 생성 -> Go embed를 통한 바이너리 통합 컴파일
make build-all

# 2. 개별 모듈 빌드
make build-cli        # CLI 도구 빌드 (bin/wuwa-tracker)
make build-server     # WebUI 포함 API 서버 빌드 (bin/wuwa-tracker-server)

# 3. 코드 포맷터, 린터 및 테스트 실행
make fmt
make lint
make test
```

---

## 사용 방법 (Usage)

### 1. WebUI 서버 실행 (`wuwa-tracker-server`)
로컬 웹 서버를 구동하여 브라우저에서 통계를 관리할 수 있습니다.

```bash
# 기본 설정(포트 3000, 로컬 DB 경로)으로 서버 구동
./bin/wuwa-tracker-server

# 커스텀 포트 및 데이터베이스 경로 설정 구동
./bin/wuwa-tracker-server -port 9090 -dbpath "./my_data/badger"
```
* 서버 구동 후 브라우저에서 `http://localhost:3000` (또는 지정한 포트)에 접속합니다.
* 복사한 튜닝 기록 URL을 입력하여 동기화하거나, JSON 백업 파일을 업로드하여 데이터를 영구 저장 및 조회할 수 있습니다.

### 2. CLI 분석 도구 실행 (`wuwa-tracker`)
터미널 환경에서 가챠 URL을 직접 입력하거나 로그 파일을 스캔하여 분석을 진행하고 보고서를 출력합니다.

```bash
# 가챠 URL을 직접 입력하여 HTML 리포트 생성
./bin/wuwa-tracker -url "https://aki-gm-resources-oversea.aki-game.net/aki/gacha/index.html?..."

# 게임 설치 폴더를 직접 스캔하여 가챠 URL 감지 및 HTML 리포트 생성
./bin/wuwa-tracker -path "<게임 설치 경로>"

# 데이터를 JSON 포맷으로 dir/out.json 파일에 저장
./bin/wuwa-tracker -url "가챠URL" -format json -out dir/out

# 데이터를 CSV 포맷으로 dir/out.csv 파일에 저장
./bin/wuwa-tracker -url "가챠URL" -format csv -out dir/out

# 데이터를 HTML 포맷으로 dir/out.html 파일에 저장
./bin/wuwa-tracker -url "가챠URL" -format html -out dir/out

# 상세 로깅 활성화 (logs 디렉터리에 가챠 기록을 JSON으로 저장합니다.)
./bin/wuwa-tracker -url "가챠URL" -v
```

#### CLI 플래그 설명
| 플래그 | 타입 | 기본값 | 설명 |
| :--- | :--- | :--- | :--- |
| `-url` | `string` | `""` | 분석할 명조 가챠 기록 URL을 입력합니다. |
| `-path` | `string` | `""` | 명조 설치 폴더를 지정하여 게임 로그 파일에서 자동으로 URL을 파싱 및 조회합니다. |
| `-format` | `string` | `"html"` | 분석 리포트의 저장 포맷을 지정합니다. (`html`, `csv`, `json` 지원) |
| `-out` | `string` | `"report"` | 생성할 리포트 파일의 이름을 지정합니다. (확장자는 포맷에 맞춰 자동 부여) |
| `-v` | `bool` | `false` | 상세 로깅(Verbose)을 활성화하며, 원격 요청 성공 시 응답받은 가챠 기록 데이터를 `logs/` 디렉터리에 타임스탬프 형식의 파일(JSON)로 자동 기록합니다. |

### 3. 오프라인 리포트 생성 (`wuwa-tracker report -f`)
```bash
# 로컬 JSON 파일에서 HTML 리포트 생성
./bin/wuwa-tracker report -f logs/my_history.json -o my_report

# 로컬 JSON 파일에서 CSV 리포트 생성
./bin/wuwa-tracker report -f logs/my_history.json -format csv -o my_report
```

---

## 프로젝트 구조 (Directory Structure)

```bash
wuwa-tracker/
├── cmd/
│   ├── cli/
│   │   └── main.go         # CLI 분석 도구 진입점 및 분석 파일 출력 파이프라인
│   └── server/
│       └── main.go         # API 웹 서버 구동, CORS 제어, 라우터 매핑, 정적 리소스 호스팅
├── config/
│   ├── config.json         # 가챠 배너별 기대 스택값, 언어 리소스, 가챠 로케일 엔드포인트 및 운 점수 임계치 정의
│   └── embed.go            # config.json의 compile-time 임베딩 관리
├── internal/
│   ├── db/
│   │   └── badger.go       # BadgerDB 제어 패키지 (플레이어 데이터 영구 보관)
│   ├── handler/
│   │   ├── errors.go       # 표준화된 HTTP 에러 응답 정의 및 관리
│   │   └── handler.go      # REST API 요청 처리 핸들러 (Sync, Upload, Stats 조회, Config 배포)
│   ├── reporter/
│   │   ├── csv.go          # CSV 포맷 평탄화 변환기
│   │   ├── exporter.go     # 익스포터 팩토리 인터페이스 정의
│   │   ├── html.go         # HTML 대시보드 빌더
│   │   └── json.go         # JSON Array 평탄화 추출기
│   ├── tracker/
│   │   ├── api.go          # Kurogame 가챠 로그 API 연동용 HTTP 클라이언트
│   │   ├── locale.go       # 외부 API 페치 실패 시 내장된 로컬 다국어 파일로 fallback 로드 수행
│   │   └── stats.go        # 가챠 통계 연산 및 픽업 사이클 기반 운 점수(Luck Score) 판별기
│   └── types/
│       └── types.go        # 공통 데이터 규격 정의
├── locales/
│   ├── ko.json             # 한국어(ko) 가챠 배너 다국어 사전 리소스 파일
│   ├── locale.go           # 임베딩된 다국어 리소스 로드 헬퍼 패키지
│   └── locale_test.go      # 다국어 리소스 로딩 단위 테스트
├── templates/
│   ├── html/
│   │   └── report.tmpl     # 리포트 HTML 대시보드 템플릿
│   └── template.go         # report.tmpl 컴파일 임베딩 패키지

├── webui/
│   ├── src/
│   │   └── App.svelte      # 대시보드 UI, JSON 업로더 및 플레이어 리스트 관리 컴포넌트
│   └── embed.go            # Svelte 빌드 결과물(dist/)의 Go static 내장 임베딩 선언
└── Makefile                # 빌드, 테스트, 포맷 빌드 자동화 스크립트
```

---

## 시스템 흐름도 (Data Sequence)

```mermaid
sequenceDiagram
    autonumber
    actor User as 사용자
    participant UI as Svelte WebUI (Browser)
    participant Server as Server (Go Fiber)
    participant DB as 로컬 DB (BadgerDB)
    participant Config as config.json
    participant API as 가챠 API

    User->>UI: 웹 화면 접속 (http://localhost:3000)
    UI->>Server: GET /api/config
    Server-->>UI: 운 점수 임계값(Thresholds) 등 설정 데이터 주입
    
    rect rgb(20, 30, 45)
        Note over User, API: [방법 A] URL을 통한 실시간 데이터 동기화
        User->>UI: 가챠 URL 입력 후 동기화 요청
        UI->>Server: POST /api/track { url }
        Server->>Config: 배너 메타데이터 로드
        Server->>API: FetchRecords() (페이지네이션 순회)
        API-->>Server: []Record (순수 데이터 목록)
        Server->>DB: SaveGachaRecords() (플레이어별 중복 없는 자동 적층)
    end

    rect rgb(30, 20, 35)
        Note over User, DB: [방법 B] 오프라인 JSON 백업 데이터 업로드
        User->>UI: JSON 파일 드래그 & 플레이어 ID 설정
        UI->>Server: POST /api/upload { playerId, data }
        Server->>DB: SaveGachaRecords() (로컬 데이터베이스에 덮어쓰기 적층)
    end

    Server->>DB: GetGachaRecords() (통계 갱신을 위해 데이터 조회)
    DB-->>Server: []Record
    Server->>Server: CalculateStats() (5성 스택, 픽업 사이클 및 운 점수 정밀 분석)
    Server-->>UI: StatsResponse { statsList } (최신 통계 렌더링용 JSON 반환)
    UI-->>User: 시각화 정보 출력 완료
```
