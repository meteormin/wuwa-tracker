# Wuwa Tracker WebUI

Svelte, TypeScript, Vite로 작성된 Wuwa Tracker 프론트엔드입니다. 빌드 결과물은 `webui/dist`에 생성되고, Go 서버 바이너리에 `go:embed`로 포함됩니다.

## 개발 명령

```bash
yarn install
yarn dev
yarn check
yarn build
```

Vite 개발 서버에서 실행할 때 API 요청과 i18n 로딩은 `http://localhost:3000`으로 전달됩니다. 따라서 개발 중에는 Go 서버도 함께 실행해야 실제 데이터를 조회할 수 있습니다.

```bash
make build
./bin/wuwa-tracker
```

## 주요 구성

- `src/App.svelte`: 전체 화면 상태, 설정 로드, 플레이어 목록, URL 동기화, JSON 업로드 흐름 관리
- `src/lib/api/index.ts`: Go 서버 API 호출 wrapper
- `src/lib/types.ts`: Go API 응답과 맞춘 프론트 타입
- `src/lib/components/ControlPanel.svelte`: URL 입력, JSON 업로드, 저장 플레이어 선택, 리포트 다운로드 UI
- `src/lib/components/GachaReport.svelte`: 배너별 통계와 Luck Score 렌더링
- `src/lib/i18n.ts`: 서버 `/api/i18n` 응답 기반 클라이언트 UI 다국어 처리

UI 번역 원본은 프론트엔드 폴더가 아니라 Go 서버의 `locales/ui/*.json`에 있습니다. 새 문구를 추가할 때는 서버 embed 리소스의 JSON key를 추가하고, 프론트에서는 `$t("key")` 형태로 참조합니다.

## API 의존성

WebUI는 다음 API를 사용합니다.

- `GET /api/config`
- `GET /api/i18n?lang=ko|en`
- `GET /api/players`
- `GET /api/stats/:playerId`
- `POST /api/track`
- `POST /api/upload`
- `GET /api/export/:playerId?format=html|json|csv&lang=ko|en`

Luck Score의 점수 구간과 상태 값은 서버 설정에서 받고, 표시 문구는 서버 embed i18n 응답으로, Tailwind 클래스 매핑은 프론트에서 `worst`, `bad`, `normal`, `good`, `best` 상태 값 기준으로 처리합니다.
