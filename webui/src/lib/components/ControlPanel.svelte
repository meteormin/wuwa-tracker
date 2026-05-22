<script lang="ts">
  import { t } from "../i18n";

  // Props 정의
  export let urlInput: string;
  export let isLoading: boolean;
  export let playersList: string[];
  export let activePlayerID: string;
  export let errorMessage: string;
  export let successMessage: string;

  // 콜백 함수 선언
  export let onTrack: () => void;
  export let onSelectPlayer: (playerId: string) => void;
  export let onFileSelect: (data: any, fileName: string) => void;
  export let onFileError: (message: string) => void;

  // 파일 입력 엘리먼트 참조 변수
  let fileInputRef: HTMLInputElement;

  // 파일 선택 이벤트 처리 함수
  function handleFileChange(event: Event) {
    const input = event.target as HTMLInputElement;
    if (!input.files || input.files.length === 0) return;

    const file = input.files[0];
    const fileName = file.name;

    const reader = new FileReader();
    reader.onload = (e) => {
      try {
        const text = e.target?.result as string;
        const jsonData = JSON.parse(text);
        onFileSelect(jsonData, fileName);
      } catch (err) {
        onFileError($t("control.invalid_json"));
      }
    };
    reader.readAsText(file);
    // 동일한 파일 재업로드 시 이벤트 정상 감지를 위해 값 리셋
    input.value = "";
  }

  // 파일 선택창 클릭 트리거 함수
  function triggerFileSelect() {
    if (fileInputRef) {
      fileInputRef.click();
    }
  }
</script>

<!-- 컨트롤 판넬 (URL 트래킹 및 유저 목록 전환) -->
<div class="glass-card p-8 mb-10 relative overflow-hidden">
  <div
    class="absolute top-0 left-0 right-0 h-[3px] bg-blue-500"
  ></div>

  <div class="grid grid-cols-1 lg:grid-cols-3 gap-6">
    <!-- 가챠 URL 입력 영역 -->
    <div class="lg:col-span-2">
      <h2
        class="text-sm font-bold text-slate-300 uppercase tracking-wider mb-2"
      >
        {$t("control.title")}
      </h2>
      <div class="flex flex-col sm:flex-row gap-2">
        <input
          type="text"
          placeholder="https://aki-gm-resources..."
          bind:value={urlInput}
          class="flex-1 bg-slate-950/80 border border-slate-800 rounded-xl px-4 py-3 text-sm focus:outline-none focus:border-blue-500 focus:ring-1 focus:ring-blue-500 transition-colors text-slate-200"
          disabled={isLoading}
        />
        <div class="flex gap-2">
          <button
            on:click={onTrack}
            class="flex-1 sm:flex-none bg-blue-600 hover:bg-blue-500 active:bg-blue-700 text-white font-bold text-sm px-6 py-3 rounded-xl transition-all shadow-md active:scale-95 disabled:opacity-50 whitespace-nowrap"
            disabled={isLoading}
          >
            {isLoading ? $t("control.tracking_btn_loading") : $t("control.tracking_btn")}
          </button>

          <!-- JSON 업로드 인풋 및 버튼 -->
          <input
            type="file"
            accept=".json"
            on:change={handleFileChange}
            bind:this={fileInputRef}
            class="hidden"
          />
          <button
            on:click={triggerFileSelect}
            class="flex-1 sm:flex-none bg-slate-850 hover:bg-slate-750 active:bg-slate-900 text-slate-200 font-bold text-sm px-5 py-3 rounded-xl transition-all border border-slate-700 flex items-center justify-center gap-1.5 active:scale-95 disabled:opacity-50 whitespace-nowrap"
            disabled={isLoading}
          >
            {$t("control.upload_btn")}
          </button>
        </div>
      </div>
    </div>

    <!-- 이전에 스캔한 유저 선택 영역 -->
    <div>
      <h2
        class="text-sm font-bold text-slate-300 uppercase tracking-wider mb-2"
      >
        {$t("control.saved_players_title", { count: playersList.length })}
      </h2>
      {#if playersList.length > 0}
        <div
          class="flex flex-wrap gap-2 max-h-[110px] overflow-y-auto custom-scrollbar p-1"
        >
          {#each playersList as player}
            <button
              on:click={() => onSelectPlayer(player)}
              class="text-xs px-3 py-1.5 font-bold rounded-lg border transition-all {activePlayerID ===
              player
                ? 'bg-blue-600/20 text-blue-400 border-blue-500'
                : 'bg-slate-900/60 text-slate-400 border-slate-800 hover:border-slate-700'}"
              disabled={isLoading}
            >
              👤 {player}
            </button>
          {/each}
        </div>
      {:else}
        <p class="text-xs text-slate-500 italic py-3">
          {$t("control.no_players")}
        </p>
      {/if}
    </div>
  </div>

  {#if activePlayerID}
    <div class="mt-6 pt-4 border-t border-slate-800/60 flex flex-wrap items-center gap-3">
      <span class="text-xs text-slate-500 font-bold uppercase tracking-wider">{$t("control.export_report")} :</span>
      <div class="flex gap-2">
        <a
          href="/api/export/{activePlayerID}?format=html"
          download="report_{activePlayerID}.html"
          class="text-xs px-3 py-1.5 font-bold rounded-lg border bg-slate-900/60 text-blue-400 border-blue-500/30 hover:border-blue-500/80 hover:bg-blue-600/10 transition-all flex items-center gap-1 active:scale-95"
        >
          📄 HTML
        </a>
        <a
          href="/api/export/{activePlayerID}?format=json"
          download="report_{activePlayerID}.json"
          class="text-xs px-3 py-1.5 font-bold rounded-lg border bg-slate-900/60 text-amber-400 border-amber-500/30 hover:border-amber-500/80 hover:bg-amber-600/10 transition-all flex items-center gap-1 active:scale-95"
        >
          📦 JSON
        </a>
        <a
          href="/api/export/{activePlayerID}?format=csv"
          download="report_{activePlayerID}.csv"
          class="text-xs px-3 py-1.5 font-bold rounded-lg border bg-slate-900/60 text-emerald-400 border-emerald-500/30 hover:border-emerald-500/80 hover:bg-emerald-600/10 transition-all flex items-center gap-1 active:scale-95"
        >
          📊 CSV
        </a>
      </div>
    </div>
  {/if}

  <!-- 피드백 알림 메시지 -->
  {#if errorMessage}
    <div
      class="mt-4 bg-rose-500/10 border border-rose-500/20 text-rose-400 rounded-xl p-3.5 text-sm flex items-center gap-2"
    >
      <span>⚠️</span>
      {errorMessage}
    </div>
  {/if}
  {#if successMessage}
    <div
      class="mt-4 bg-emerald-500/10 border border-emerald-500/20 text-emerald-400 rounded-xl p-3.5 text-sm flex items-center gap-2"
    >
      <span>✅</span>
      {successMessage}
    </div>
  {/if}
</div>
