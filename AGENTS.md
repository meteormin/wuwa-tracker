# Wuwa Tracker

## Architecture

- cmd: entry point
- internal: backend api server 및 비즈니스로직, 서버 기반은 gofiber 사용
- webui: frontend, svelte 사용

## Global Rules

1. 코드 내, 주석 제외 한글 사용 금지
2. CGO 사용 금지
3. 기타 외부 통신 라이브러리 사용 금지 (http client 제외)

## Build Guide

- backend server는 `Makefile` 및 `go build`를 사용
- frontend webui는 `yarn`, `vite`를 사용
- 빌드 관련 커맨드는 [Makefile](./Makefile)을 참조

## Codebase Rules

Go 언어에서 권장하는 표준을 준수

- Linter: golangci-lint
- LSP: gopls
- Formatter: gofumpt

### Comments

- 코드 주석은 한글로 작성합니다. 에러 메시지, 로그/콘솔 출력은 제외합니다.
- 고유명사 혹은 한글로 번역 시 오히려 의미 해석이 어색해지는 경우는 판단하에 영어로 작성합니다.
- 코멘트는 간결하게 핵심만 작성합니다.
- 성능 및 구현 공수 때문에 복잡하거나 가독성이 떨어지는 경우는 상세하게 코멘트를 작성합니다.

### Branch Strategy

- 기본 브랜치는 `main`으로 유지합니다.
- `main`은 릴리즈 가능한 안정 브랜치로 관리합니다.
- `develop`은 다음 릴리즈 후보를 통합하는 상시 개발 브랜치로 유지합니다.
- 일반 기능 작업은 `feature/<description>` 브랜치에서 진행하고 `develop`으로 PR을 생성합니다.
- 버그 수정은 `fix/<description>` 브랜치에서 진행합니다.
- 긴급 수정이 필요한 경우 `main`에서 `hotfix/<description>` 브랜치를 만들고, 머지 후 `develop`에도 반영합니다.
- `main` 머지는 릴리즈 생성 기준이므로 직접 push 대신 PR을 사용합니다.

### Commit Message

- 커밋 메시지는 Conventional Commits 형식을 사용합니다.
- 메시지 형식은 `<type>: <summary>`를 기본으로 합니다.
- 기능 추가는 `feat`, 버그 수정은 `fix`, CI/워크플로우 변경은 `ci`, 문서 변경은 `docs`, 테스트 변경은 `test`, 리팩터링은 `refactor`, 기타 관리 작업은 `chore`를 사용합니다.
- 브랜치 성격과 변경 내용을 기준으로 적절한 prefix를 선택합니다.
- 예시:
  - `feat: centralize runtime config defaults`
  - `fix: prepare embedded webui before CI tests`
  - `ci: add release workflow`

## Workflows

### Documentation Analysis and DESIGN.md Synthesis

이 워크플로우의 목적은 프로젝트의 기존 문서, 주석, 그리고 실제 코드를 단계적으로 분석하여 실제 구현과 일치하는 최신화된 `DESIGN.md` 파일을 생성하거나 수정하는 것입니다.

#### Analysis Principles

- Source of Truth: 문서와 코드가 상충할 경우, 실제 동작하는 코드를 최우선 순위로 둡니다.
- Incremental Update: 기존 `DESIGN.md`가 존재할 경우, 무조건적인 덮어쓰기보다 변경 사항을 추적하여 논리적으로 수정합니다.
- Evidence-based: 모든 설명은 프로젝트 내의 물리적 파일 경로와 연결되어야 합니다.

#### Execution Steps

1. Document Analysis
   - 대상: 프로젝트 루트 및 `docs/` 디렉토리 내의 모든 `.md` 파일.
   - 목표: 프로젝트의 설계 의도, 비즈니스 로직의 배경, 아키텍처 가이드를 파악합니다.
   - 결과: 가상의 설계 지도를 생성합니다.

2. Spec Extraction
   - 대상: 소스 코드 전체 (`src/` 등).
   - 목표: 클래스, 메서드 상단에 기술된 Javadoc, Docstring 등을 추출합니다.
   - 비교: Step 1에서 파악한 설계 의도가 실제 인터페이스 명세(주석)와 일치하는지 대조합니다.

3. Code Deep-Dive
   - 대상: 실제 비즈니스 로직 구현체 (`.java`, `.py`, `.ts`, `.go` 등).
   - 목표: 실제 호출 흐름, 데이터 저장 방식, 예외 처리 로직을 분석합니다.
   - 대조 검증:
     - 문서에는 존재하지만 코드에는 없는 기능 식별.
     - 코드에는 구현되었으나 문서/주석에 누락된 로직 식별.
     - 기존 분석 결과와 실제 코드 간의 논리적 모순 기록.

4. DESIGN.md Synthesis
   - 대상: 프로젝트 루트의 `DESIGN.md`.
   - 구성:
     - Architecture Overview: 프로젝트의 전체적인 구조.
     - Component/Module Details: 분석된 실제 모듈별 역할 정의.
     - Implementation vs Design: 문서와 실제 코드 간의 주요 차이점이 있는 경우 기록.
     - Updated Date: 분석 완료 시간 및 기준 커밋 정보가 가능할 경우 기록.

#### Documentation Agent Requirements

- 분석 중 발견된 모든 불일치(Discrepancy)는 무시하지 말고 `DESIGN.md`에 Notes 또는 Known Limitations 섹션으로 기록할 것.
- 가급적 표준 기술 용어(Design Patterns, Architecture Styles)를 사용하여 전문성을 유지할 것.
- 출력물(`DESIGN.md`)의 언어 설정은 사용자의 별도 지시가 없다면 프로젝트의 주 사용 언어를 따를 것.

### Commit Workflow

이 워크플로우는 모든 현재 변경 사항을 스테이징하고, 설명적인 커밋 메시지를 생성한 뒤 커밋하는 절차입니다.

1. Audit Packages
   ```bash
   make audit
   ```

2. Build Check
   ```bash
   go build ./...
   ```

3. Stage All Changes
   ```bash
   git add .
   ```

4. Analyze Changes
   ```bash
   git diff --cached
   ```

5. Generate and Commit
   - Conventional Commits 형식의 전문적인 메시지를 생성하여 커밋합니다.
   ```bash
   git commit -m "<ai_generated_message>"
   ```

6. Push
   - 필요할 경우 변경 사항을 push합니다.
   ```bash
   git push
   ```

7. Prepare PR Description
   - PR을 생성할 경우 PR 본문에 붙여 넣을 설명을 생성하고 출력합니다.
   - `Summary` 섹션은 PR의 전체 의도를 간결하게 설명합니다.
   - `Changes` 섹션은 구체적인 구현 변경 사항을 작성합니다.
   - `Verification` 섹션은 확인한 명령을 포함합니다.
   - 의미 있는 커밋 단위로 나눈 경우 `Commits`를 포함합니다.
   - 호환성, 리뷰 집중 영역, 마이그레이션, 릴리즈 고려 사항이 있을 때만 `Review Notes`를 포함합니다.
   - 각 bullet은 집중된 내용으로 작성하고 긴 서술형 문단은 피합니다.

   ```markdown
   ## Summary

   <One short paragraph explaining the purpose of the PR.>

   ## Changes

   - ...

   ## Commits

   - `<commit message>`

   ## Verification

   - [x] `...`

   ## Review Notes

   - ...
   ```
