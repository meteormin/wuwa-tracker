# Wuwa Tracker

## Architecture

- cmd: entry point
- internal: backend api server 및 비즈니스로직, 서버 기반은 gofiber 사용
- webui: frontend, svelte 사용

## 주의 사항

1. 코드 내, 주석 제외 한글 사용 금지
2. CGO 사용 금지
3. 기타 외부 통신 라이브러리 사용 금지 (http client 제외)
