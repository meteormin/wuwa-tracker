# 명조: 워더링 웨이브 - 튜닝 통계 트래커 (Wuwa Tracker CLI)

명조: 워더링 웨이브(Wuthering Waves)의 튜닝(가챠) 기록 URL을 분석하여 누적 스택, 천장(Pity) 계산, 획득한 5성 캐릭터/무기 히스토리를 요약하고, 이를 프리미엄 인터랙티브 HTML 대시보드와 데이터 분석용 CSV/JSON 포맷으로 출력하는 Go 기반 CLI 도구입니다.

---

## 🚀 주요 기능 및 특징

1. **완벽한 다국어 지원 및 시스템 로케일 Fallback**
   * 게임 기록 URL에서 언어(`lang`) 파라미터를 자동 추출하여 다국어 번역을 적용합니다.
   * URL에 언어 파라미터가 누락된 경우, OS 시스템 환경 변수(`LC_ALL`, `LANG`)를 파싱하여 적절한 시스템 로케일(예: `ko_KR` -> `ko`)을 자동으로 추적하며, 최종 실패 시 `"ko"`로 안전하게 대체됩니다.
2. **동적 가챠 구성 시스템 (Hardcoding Zero)**
   * 가챠 배너 정보나 5성 상시 목록이 코드 내에 하드코딩되어 있지 않습니다.
   * 임베드된 `config.json` 설정 파일과 공식 게임 리소스 서버의 다국어 번역 리소스(`locales/{lang}.json`)를 실시간으로 다운로드하여 완전히 동적으로 매핑합니다.
3. **평탄화(Flattened) 데이터 내보내기**
   * **CSV/JSON**: 획득 이력이 3★, 4★, 5★ 전체 기록 단위로 가챠 유형 및 가챠 명칭 정보와 Join되어 1차원으로 평탄화(Flatten)된 상태로 저장되므로, 엑셀이나 외부 통계 툴에서 즉시 데이터 분석을 시작할 수 있습니다.
4. **프리미엄 인터랙티브 HTML 대시보드**
   * 시각적으로 높은 만족도를 주는 모던 다크 테마 대시보드를 제공합니다.
   * **상호작용형 운 지수(Luck Index) 계산기**: 사용자가 웹 브라우저에서 실시간으로 각 배너의 "평균 기대 뽑기 횟수" 및 "기본 확률"을 수정하면 실시간으로 운 상태(예: '비정상적인 행운! 🔥', '극악의 억까 상태... 💀') 및 지수(%)를 동적으로 재계산합니다.
   * **접기/펼치기 전체 히스토리**: 대량의 뽑기 목록을 등급별 색상 배지와 함께 콤팩트한 스크롤 드로어 컴포넌트로 축소하여 브라우저 과부하 없이 편안한 스크롤 조회를 제공합니다.

---

## 🛠️ 개발 환경 및 빌드 방법

### 개발 환경 (Prerequisites)
* **Go**: 1.20 버전 이상 권장
* **CGO 미사용**: `CGO_ENABLED=0` 환경에서 순수 Go로만 빌드됩니다.
* **의존성 규격**: `golangci-lint`, `gofumpt` 코드 규격을 준수합니다.

### 빌드 및 실행 명령어

제공되는 [Makefile](file:///Users/yooseongmin/Projects/meteormin/wuwa-tracker/Makefile)을 통해 간단하게 빌드하고 실행할 수 있습니다.

```bash
# 1. 의존성 다운로드 및 포맷팅
make fmt

# 2. 코드 린터 및 정적 분석 실행
make lint

# 3. 단위 테스트 수행
make test

# 4. 프로젝트 빌드 (bin/wuwa-tracker 바이너리 파일 생성)
make build

# 5. CLI 즉시 빌드 및 실행
make run
```

---

## 💻 사용법

빌드된 실행 파일을 실행하거나 `go run` 명령어를 통해 실행 시 Wuthering Waves의 **가챠 기록 URL**을 인자로 전달합니다.

```bash
# 빌드된 바이너리 실행 예시 (가챠 URL 입력)
./bin/wuwa-tracker "https://aki-gm-resources-oversea.aki-game.net/aki/gacha/index.html?..."
```

실행이 완료되면 루트 경로에 다음 파일들이 생성됩니다:
* `report.html`: 웹 브라우저로 열 수 있는 반응형 인터랙티브 튜닝 리포트 대시보드
* `report.csv`: 엑셀 분석용 1차원 평탄화 데이터 파일
* `report.json`: 다른 애플리케이션 연동을 위한 JSON 포맷 파일

---

## 📂 프로젝트 폴더 구조

```bash
wuwa-tracker/
├── cmd/
│   └── cli/
│       └── main.go         # CLI 진입점 및 전체 프로세스 오케스트레이션
├── config/
│   ├── config.json         # 가챠 배너 메타데이터 및 5성 상시 목록 정의
│   └── embed.go            # config.json의 compile-time 임베딩 패키지
├── internal/
│   ├── reporter/
│   │   ├── csv.go          # 평탄화된 1차원 CSV 포맷 내보내기 구현
│   │   ├── json.go         # 평탄화된 JSON Array 포맷 내보내기 구현
│   │   ├── html.go         # go:embed 템플릿 기반 HTML 대시보드 익스포터
│   │   └── exporter.go     # Exporter 공통 인터페이스 규격 정의
│   ├── tracker/
│   │   ├── api.go          # 게임사 공식 API 요청 및 다국어 locales 패치 처리
│   │   ├── stats.go        # 누적 스택, 5성 픽업/픽뚫 여부 핵심 연산 로직
│   │   └── stats_test.go   # 가챠 스택 계산 알고리즘 단위 검증 테스트
│   └── types/
│       └── types.go        # 도메인 코어 데이터 모델 및 구조체 정의
├── templates/
│   ├── html/
│   │   └── report.tmpl     # Tailwind CSS & Luck Index JS 탑재 대시보드 템플릿
│   └── template.go         # report.tmpl의 compile-time 임베딩 패키지
└── Makefile                # 빌드, 포맷팅, 테스트 오토메이션 스크립트
```

---

## 🔄 시스템 흐름도 (Sequence Diagram)

전체 수집 및 분석 프로세스의 상세 메커니즘 흐름은 다음과 같습니다.

```mermaid
sequenceDiagram
    autonumber
    actor User as 사용자 (CLI)
    participant CLI as cmd/cli/main.go
    participant Config as config (config.json)
    participant API as Wuthering Waves API
    participant Calc as internal/tracker (StatsCalculator)
    participant Exp as internal/reporter (CSV/JSON/HTML)

    User->>CLI: 실행 (가챠 기록 URL 입력)
    CLI->>CLI: URL 파싱 및 언어(lang) 추출
    Note over CLI: lang 파라미터가 없으면 OS 로케일 감지 (Fallback: "ko")
    
    CLI->>Config: config.LoadConfig() (배너 및 5성 상시 목록 로드)
    CLI->>API: FetchGachaLocale(lang) (공식 번역 텍스트 조회)
    API-->>CLI: locales/{lang}.json (가챠 유형 이름 매핑 데이터)
    
    loop 각 가챠 유형별 (GachaTypes 순회)
        CLI->>API: FetchGachaRecords(Page 1..N) (가챠 기록 페이지네이션 조회)
        API-->>CLI: []Record (가챠 이력 목록)
    end
    
    CLI->>Calc: CalculateStats([]Record, GachaType) (스택 및 5성 획득 정보 계산)
    Calc-->>CLI: types.Stats (5성 목록, 4성/5성 천장 피티 스택 등)
    
    CLI->>Exp: Export([]types.Stats, OutputPath) (리포트 출력)
    Note over Exp: HTML (인터랙티브 대시보드)<br/>CSV/JSON (조인하여 평탄화된 데이터 포맷)
    Exp-->>User: report.html, report.csv, report.json 생성 완료
```
