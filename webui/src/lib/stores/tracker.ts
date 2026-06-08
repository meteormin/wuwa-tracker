import { get, writable } from "svelte/store";
import type { LuckScoreThreshold, Stats } from "../types";
import {
  fetchConfig,
  fetchPlayers,
  fetchStats,
  scanURL as apiScanURL,
  trackURL as apiTrackURL,
  uploadJSON,
} from "../api";
import { initI18n, t } from "../i18n";

export const urlInput = writable("");
export const scanPathInput = writable("");
export const isLoading = writable(false);
export const isScanning = writable(false);
export const errorMessage = writable("");
export const successMessage = writable("");
export const activePlayerID = writable("");
export const playersList = writable<string[]>([]);
export const activeStats = writable<Stats[]>([]);
export const thresholds = writable<LuckScoreThreshold[]>([]);

function translate(key: string, replaceParams?: Record<string, string | number>) {
  return get(t)(key, replaceParams);
}

function apiErrorMessage(
  data: { error?: string; errorKey?: string },
  fallbackKey: string,
) {
  return data.errorKey ? translate(data.errorKey) : data.error || translate(fallbackKey);
}

function clearMessages() {
  errorMessage.set("");
  successMessage.set("");
}

export async function loadConfig() {
  try {
    const data = await fetchConfig();
    if (data.success && data.luckScoreThresholds) {
      thresholds.set(data.luckScoreThresholds);
    }
  } catch (e) {
    console.warn("Failed to load server configuration, using local defaults:", e);
  }
}

export async function loadPlayers() {
  try {
    const data = await fetchPlayers();
    if (data.success) {
      playersList.set(data.players || []);
    }
  } catch (e) {
    console.error("Failed to load tracked player history list:", e);
  }
}

export async function selectPlayer(playerId: string) {
  isLoading.set(true);
  clearMessages();
  activePlayerID.set(playerId);

  try {
    const data = await fetchStats(playerId);
    if (data.success) {
      activeStats.set(data.stats);
    } else {
      errorMessage.set(apiErrorMessage(data, "app.failed_stats"));
    }
  } catch (e) {
    errorMessage.set(translate("app.network_error"));
  } finally {
    isLoading.set(false);
  }
}

export async function scanURL() {
  const scanPath = get(scanPathInput).trim();
  if (!scanPath) {
    errorMessage.set(translate("app.scan_path_empty_alert"));
    successMessage.set("");
    return;
  }

  isLoading.set(true);
  isScanning.set(true);
  clearMessages();

  try {
    const data = await apiScanURL(scanPath);
    if (data.success) {
      urlInput.set(data.url);
      successMessage.set(translate("app.scan_success"));
    } else {
      errorMessage.set(apiErrorMessage(data, "app.failed_scan"));
    }
  } catch (e) {
    errorMessage.set(translate("app.network_error"));
  } finally {
    isScanning.set(false);
    isLoading.set(false);
  }
}

export async function trackURL() {
  const cleanURL = get(urlInput).trim();
  if (!cleanURL) {
    errorMessage.set(translate("app.url_empty_alert"));
    return;
  }

  isLoading.set(true);
  clearMessages();

  try {
    const data = await apiTrackURL(cleanURL);
    if (data.success) {
      activePlayerID.set(data.playerId);
      activeStats.set(data.stats);
      successMessage.set(translate("app.track_success", { playerId: data.playerId }));
      urlInput.set("");
      await loadPlayers();
    } else {
      errorMessage.set(apiErrorMessage(data, "app.failed_scan"));
    }
  } catch (e) {
    errorMessage.set(translate("app.network_error"));
  } finally {
    isLoading.set(false);
  }
}

export async function handleFileSelect(data: any, fileName: string) {
  const playerId = data.payload?.playerId?.trim();
  if (!playerId) {
    errorMessage.set(translate("app.failed_upload"));
    return;
  }

  isLoading.set(true);
  clearMessages();

  try {
    const res = await uploadJSON(data);
    if (res.success) {
      activePlayerID.set(res.playerId);
      activeStats.set(res.stats);
      successMessage.set(
        translate("app.upload_success", {
          fileName,
          playerId: res.playerId,
        }),
      );
      await loadPlayers();
    } else {
      errorMessage.set(apiErrorMessage(res, "app.failed_upload"));
    }
  } catch (e) {
    errorMessage.set(translate("app.failed_upload_network"));
  } finally {
    isLoading.set(false);
  }
}

export function handleFileError(message: string) {
  errorMessage.set(message);
  successMessage.set("");
}

export async function initializeTracker() {
  try {
    await initI18n();
  } catch (e) {
    console.warn("Failed to initialize translations:", e);
  }

  await loadConfig();
  await loadPlayers();

  const players = get(playersList);
  if (players.length > 0) {
    await selectPlayer(players[0]);
  }
}
