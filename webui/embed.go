package webui

import (
	"embed"
	"io/fs"
)

// dist 는 Vite + Svelte 빌드로 생성된 dist 폴더의 정적 파일들을 내장(Embed)합니다.
//
//go:embed dist
var dist embed.FS

var FS, _ = fs.Sub(dist, "dist")
