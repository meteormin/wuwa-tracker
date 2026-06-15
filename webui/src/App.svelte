<script lang="ts">
  import { onMount } from "svelte";
  import {
    activePlayerID,
    activeStats,
    errorMessage,
    exportReport,
    handleFileError,
    handleFileSelect,
    initializeTracker,
    isLoading,
    isScanning,
    playersList,
    scanPathInput,
    scanURL,
    selectPlayer,
    successMessage,
    thresholds,
    trackURL,
    urlInput,
  } from "./lib/stores/tracker";
  import { t } from "./lib/i18n";
  import Header from "./lib/components/Header.svelte";
  import ControlPanel from "./lib/components/ControlPanel.svelte";
  import GachaReport from "./lib/components/GachaReport.svelte";

  onMount(initializeTracker);
</script>

<div class="max-w-7xl mx-auto px-6 md:px-12 py-10 md:py-16">
  <!-- 헤더 -->
  <Header />

  <!-- 컨트롤 판넬 (URL 트래킹 및 유저 목록 전환) -->
  <ControlPanel
    bind:urlInput={$urlInput}
    bind:scanPathInput={$scanPathInput}
    isLoading={$isLoading}
    isScanning={$isScanning}
    playersList={$playersList}
    activePlayerID={$activePlayerID}
    errorMessage={$errorMessage}
    successMessage={$successMessage}
    onScan={scanURL}
    onTrack={trackURL}
    onSelectPlayer={selectPlayer}
    onFileSelect={handleFileSelect}
    onFileError={handleFileError}
    onExport={exportReport}
  />

  <!-- 통계 리포트 뷰어 영역 -->
  {#if $activeStats.length > 0}
    <div class="space-y-16">
      {#each $activeStats as stat}
        <GachaReport {stat} thresholds={$thresholds} />
      {/each}
    </div>
  {:else if !$isLoading}
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
