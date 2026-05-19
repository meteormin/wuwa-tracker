<script lang="ts">
  import { onMount } from "svelte";
  import type { Stats, LuckScoreThreshold } from "./lib/types";
  import {
    fetchConfig,
    fetchPlayers,
    fetchStats,
    trackURL as apiTrackURL,
    uploadJSON,
  } from "./lib/api";
  import { t } from "./lib/i18n";
  import Header from "./lib/components/Header.svelte";
  import ControlPanel from "./lib/components/ControlPanel.svelte";
  import UploadModal from "./lib/components/UploadModal.svelte";
  import GachaReport from "./lib/components/GachaReport.svelte";

  // 상태값 선언
  let urlInput = "";
  let isLoading = false;
  let errorMessage = "";
  let successMessage = "";
  let activePlayerID = "";
  let playersList: string[] = [];
  let activeStats: Stats[] = [];

  // 오프라인 테스트용 JSON 업로드 상태 정의
  let showUploadModal = false;
  let uploadPlayerID = "";
  let uploadedJSONData: any = null;
  let uploadFileName = "";

  // 운 점수 임계치 정의 (서버 설정을 주입받기 전의 기본 폴백값)
  let thresholds: LuckScoreThreshold[] = [];

  // 서버의 가챠 리포트 설정을 유연하게 로드
  async function loadConfig() {
    try {
      const data = await fetchConfig();
      if (data.success && data.luckScoreThresholds) {
        thresholds = data.luckScoreThresholds;
      }
    } catch (e) {
      console.warn(
        "Failed to load server configuration, using local defaults:",
        e,
      );
    }
  }

  // 저장된 플레이어 목록 로드
  async function loadPlayers() {
    try {
      const data = await fetchPlayers();
      if (data.success) {
        playersList = data.players || [];
      }
    } catch (e) {
      console.error("Failed to load tracked player history list:", e);
    }
  }

  // 플레이어 선택 시 로컬 SQLite 데이터 즉각 조회
  async function selectPlayer(playerId: string) {
    isLoading = true;
    errorMessage = "";
    successMessage = "";
    activePlayerID = playerId;
    try {
      const data = await fetchStats(playerId);
      if (data.success) {
        activeStats = data.stats;
      } else {
        errorMessage = data.errorKey ? $t(data.errorKey as any) : (data.error || $t("app.failed_stats"));
      }
    } catch (e) {
      errorMessage = $t("app.network_error");
    } finally {
      isLoading = false;
    }
  }

  // Kurogame 가챠 결과 원격 스캔 & 트래킹 등록
  async function trackURL() {
    let cleanURL = urlInput.trim();
    if (!cleanURL) {
      errorMessage = $t("app.url_empty_alert");
      return;
    }

    isLoading = true;
    errorMessage = "";
    successMessage = "";

    try {
      const data = await apiTrackURL(cleanURL);
      if (data.success) {
        activePlayerID = data.playerId;
        activeStats = data.stats;
        successMessage = $t("app.track_success", { playerId: data.playerId });
        urlInput = "";
        // 신규 유저 히스토리 갱신
        await loadPlayers();
      } else {
        errorMessage = data.errorKey ? $t(data.errorKey as any) : (data.error || $t("app.failed_scan"));
      }
    } catch (e) {
      errorMessage = $t("app.network_error");
    } finally {
      isLoading = false;
    }
  }

  // 파일 분석 성공 콜백
  function handleFileSelect(data: any, fileName: string) {
    uploadedJSONData = data;
    uploadFileName = fileName;
    const baseName = fileName.replace(/\.[^/.]+$/, "");
    uploadPlayerID = baseName;
    errorMessage = "";
    successMessage = "";
    showUploadModal = true;
  }

  // 파일 분석 실패 콜백
  function handleFileError(message: string) {
    errorMessage = message;
    successMessage = "";
    uploadedJSONData = null;
    uploadFileName = "";
  }

  // JSON 업로드 최종 제출
  async function submitJSONUpload() {
    const cleanID = uploadPlayerID.trim();
    if (!cleanID) {
      window.alert($t("app.player_id_empty"));
      return;
    }

    if (!uploadedJSONData) {
      window.alert($t("app.json_empty"));
      return;
    }

    isLoading = true;
    errorMessage = "";
    successMessage = "";
    showUploadModal = false;

    try {
      const data = await uploadJSON(cleanID, uploadedJSONData);
      if (data.success) {
        activePlayerID = data.playerId;
        activeStats = data.stats;
        successMessage = $t("app.upload_success", {
          fileName: uploadFileName,
          playerId: data.playerId,
        });
        uploadedJSONData = null;
        uploadFileName = "";
        // 신규 유저 히스토리 갱신
        await loadPlayers();
      } else {
        errorMessage = data.errorKey ? $t(data.errorKey as any) : (data.error || $t("app.failed_upload"));
      }
    } catch (e) {
      errorMessage = $t("app.failed_upload_network");
    } finally {
      isLoading = false;
    }
  }

  // JSON 업로드 취소 콜백
  function cancelJSONUpload() {
    showUploadModal = false;
    uploadedJSONData = null;
    uploadFileName = "";
  }

  onMount(async () => {
    await loadConfig();
    await loadPlayers();
    // 데이터가 이미 저장된 기존 첫 번째 유저가 있다면 자동으로 선로딩 수행
    if (playersList.length > 0) {
      await selectPlayer(playersList[0]);
    }
  });
</script>

<div class="max-w-7xl mx-auto px-6 md:px-12 py-10 md:py-16">
  <!-- 헤더 -->
  <Header />

  <!-- 컨트롤 판넬 (URL 트래킹 및 유저 목록 전환) -->
  <ControlPanel
    bind:urlInput
    {isLoading}
    {playersList}
    {activePlayerID}
    {errorMessage}
    {successMessage}
    onTrack={trackURL}
    onSelectPlayer={selectPlayer}
    onFileSelect={handleFileSelect}
    onFileError={handleFileError}
  />

  <!-- 통계 리포트 뷰어 영역 -->
  {#if activeStats.length > 0}
    <div class="space-y-16">
      {#each activeStats as stat}
        <GachaReport {stat} {thresholds} />
      {/each}
    </div>
  {:else if !isLoading}
    <div class="glass-card p-12 text-center text-slate-500 border-dashed">
      <span class="text-4xl block mb-3">🔮</span>
      <p class="text-base font-bold text-slate-300 mb-1">
        {$t("app.no_data")}
      </p>
      <p class="text-xs">
        {$t("app.no_data_desc")}
      </p>
    </div>
  {/if}
</div>

<!-- JSON 업로드용 모달 오버레이 -->
<UploadModal
  show={showUploadModal}
  {uploadFileName}
  bind:uploadPlayerID
  onCancel={cancelJSONUpload}
  onSubmit={submitJSONUpload}
/>
