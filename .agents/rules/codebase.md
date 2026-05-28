---
trigger: always_on
---

# Codebase

Go 언어에서 권장하는 표준을 준수

- Linter: golangci-lint
- LSP: gopls
- Formatter: gofumpt

## Comments
- 코드 주석은 한글로 작성합니다.(에러 메시지, 로그/콘솔 출력은 제외)
- 고유명사 혹은 한글로 번역 시 오히려 의미 해석이 어색해지는 경우는 판단하에 영어로 작성
- 코멘트는 간결하게 핵심만 작성합니다.
- 성능 및 구현 공수 때문에 복잡하거나 가독성이 떨어지는 경우는 상세하게 코멘트를 작성합니다.

## Branch Strategy
- 기본 브랜치는 `main`으로 유지합니다.
- `main`은 릴리즈 가능한 안정 브랜치로 관리합니다.
- `develop`은 다음 릴리즈 후보를 통합하는 상시 개발 브랜치로 유지합니다.
- 일반 기능 작업은 `feature/<description>` 브랜치에서 진행하고 `develop`으로 PR을 생성합니다.
- 버그 수정은 `fix/<description>` 브랜치에서 진행합니다.
- 긴급 수정이 필요한 경우 `main`에서 `hotfix/<description>` 브랜치를 만들고, 머지 후 `develop`에도 반영합니다.
- `main` 머지는 릴리즈 생성 기준이므로 직접 push 대신 PR을 사용합니다.

## Commit Message
- 커밋 메시지는 Conventional Commits 형식을 사용합니다.
- 메시지 형식은 `<type>: <summary>`를 기본으로 합니다.
- 기능 추가는 `feat`, 버그 수정은 `fix`, CI/워크플로우 변경은 `ci`, 문서 변경은 `docs`, 테스트 변경은 `test`, 리팩터링은 `refactor`, 기타 관리 작업은 `chore`를 사용합니다.
- 브랜치 성격과 변경 내용을 기준으로 적절한 prefix를 선택합니다.
- 예시:
  - `feat: centralize runtime config defaults`
  - `fix: prepare embedded webui before CI tests`
  - `ci: add release workflow`
