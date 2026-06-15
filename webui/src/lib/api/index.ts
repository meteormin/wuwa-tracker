import type { Stats, LuckScoreThreshold } from "../types";
import { invoke } from "@tauri-apps/api/core";
import { apiHost, isTauriRuntime } from "./config";

// API 표준 공통 응답 구조
export interface BaseResponse {
  success: boolean;
  error?: string;
  errorKey?: string;
}

// 가챠 설정 API 응답 인터페이스
export interface ConfigResponse extends BaseResponse {
  luckScoreThresholds?: LuckScoreThreshold[];
}

// 플레이어 목록 API 응답 인터페이스
export interface PlayersResponse extends BaseResponse {
  players?: string[];
}

// 가챠 통계 API 응답 인터페이스
export interface StatsResponse extends BaseResponse {
  stats: Stats[];
}

// 가챠 트래킹 API 응답 인터페이스
export interface TrackResponse extends BaseResponse {
  playerId: string;
  stats: Stats[];
}

// 로컬 로그 스캔 API 응답 인터페이스
export interface ScanResponse extends BaseResponse {
  url: string;
}

// JSON 업로드 API 응답 인터페이스
export interface UploadResponse extends BaseResponse {
  playerId: string;
  stats: Stats[];
}

export interface ExportResponse extends BaseResponse {
  filename: string;
  contentType: string;
  content: number[];
}

/**
 * 서버의 가챠 리포트 설정을 유연하게 로드합니다.
 */
export async function fetchConfig(): Promise<ConfigResponse> {
  if (isTauriRuntime()) {
    return invoke<ConfigResponse>("get_config");
  }
  const res = await fetch(`${apiHost}/api/config`);
  return res.json();
}

/**
 * 저장된 플레이어 목록을 조회합니다.
 */
export async function fetchPlayers(): Promise<PlayersResponse> {
  if (isTauriRuntime()) {
    return invoke<PlayersResponse>("list_players");
  }
  const res = await fetch(`${apiHost}/api/players`);
  return res.json();
}

/**
 * 플레이어 ID에 해당하는 가챠 통계 데이터를 조회합니다.
 */
export async function fetchStats(playerId: string): Promise<StatsResponse> {
  if (isTauriRuntime()) {
    return invoke<StatsResponse>("get_stats", { playerId });
  }
  const res = await fetch(`${apiHost}/api/stats/${playerId}`);
  return res.json();
}

/**
 * 로컬 게임 로그 경로에서 가챠 URL 스캔을 요청합니다.
 */
export async function scanURL(path: string): Promise<ScanResponse> {
  if (isTauriRuntime()) {
    return invoke<ScanResponse>("scan_url", { path });
  }
  const res = await fetch(`${apiHost}/api/scan`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify({ path }),
  });
  return res.json();
}

/**
 * Kurogame 가챠 결과 원격 스캔 및 트래킹 등록을 요청합니다.
 */
export async function trackURL(url: string): Promise<TrackResponse> {
  if (isTauriRuntime()) {
    return invoke<TrackResponse>("track_url", { url });
  }
  const res = await fetch(`${apiHost}/api/track`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify({ url }),
  });
  return res.json();
}

/**
 * 오프라인 분석을 위한 JSON 파일 데이터를 서버에 업로드합니다.
 */
export async function uploadJSON(data: any): Promise<UploadResponse> {
  if (isTauriRuntime()) {
    return invoke<UploadResponse>("upload_json", { data });
  }
  const res = await fetch(`${apiHost}/api/upload`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify(data),
  });
  return res.json();
}

export async function exportReport(
  playerId: string,
  format: "html" | "json" | "csv",
  lang: string,
): Promise<void> {
  if (isTauriRuntime()) {
    const data = await invoke<ExportResponse>("export_report", {
      playerId,
      format,
      lang,
    });
    const blob = new Blob([new Uint8Array(data.content)], {
      type: data.contentType,
    });
    downloadBlob(blob, data.filename);
    return;
  }

  const res = await fetch(`${apiHost}/api/export/${playerId}?format=${format}&lang=${lang}`);
  if (!res.ok) {
    throw new Error(`failed to export report: ${res.status}`);
  }
  const blob = await res.blob();
  downloadBlob(blob, `report_${playerId}.${format}`);
}

function downloadBlob(blob: Blob, filename: string) {
  const url = URL.createObjectURL(blob);
  const link = document.createElement("a");
  link.href = url;
  link.download = filename;
  document.body.appendChild(link);
  link.click();
  link.remove();
  URL.revokeObjectURL(url);
}
