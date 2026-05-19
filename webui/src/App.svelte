<script lang="ts">
  import { onMount } from "svelte";

  // 인터페이스 정의
  interface FiveStarRecord {
    name: string;
    time: string;
    pity: number;
    isPickUp: boolean;
  }

  interface Record {
    cardPoolType: string;
    resourceId: number;
    qualityLevel: number;
    resourceType: string;
    name: string;
    count: number;
    time: string;
  }

  interface Stats {
    gachaType: number;
    gachaName: string;
    totalPulls: number;
    currentPity5: number;
    currentPity4: number;
    baseRate: number;
    expectedPulls: number;
    avgPulls: number;
    actualRate: number;
    luckScore: number;
    fiveStars: FiveStarRecord[] | null;
    records: Record[] | null;
    hasFiveStar: boolean;
  }

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
  let fileInputRef: HTMLInputElement;

  // 호스트 자동 감지 (Vite 개발 모드 시 8080 포트로 라우팅)
  const apiHost = import.meta.env.DEV ? "http://localhost:8080" : "";

  interface LuckScoreThreshold {
    minScore: number;
    state: string;
    colorClass: string;
    bgClass: string;
  }

  // 운 점수 임계치 정의 (서버 설정을 주입받기 전의 기본 폴백값)
  let thresholds: LuckScoreThreshold[] = [
    {
      minScore: 0.0,
      state: "WORST",
      colorClass: "text-rose-500 font-extrabold",
      bgClass: "bg-rose-500/10 border-rose-500/30",
    },
    {
      minScore: 85.0,
      state: "BAD",
      colorClass: "text-rose-300",
      bgClass: "bg-rose-500/5 border-rose-500/10",
    },
    {
      minScore: 95.0,
      state: "NORMAL",
      colorClass: "text-slate-300",
      bgClass: "bg-slate-900/40 border-slate-800/50",
    },
    {
      minScore: 102.0,
      state: "LUCKY",
      colorClass: "text-emerald-400",
      bgClass: "bg-emerald-500/5 border-emerald-500/20",
    },
    {
      minScore: 115.0,
      state: "LUCKIEST",
      colorClass: "text-emerald-400 animate-pulse",
      bgClass: "bg-emerald-500/10 border-emerald-500/30",
    },
  ];

  // 서버의 가챠 리포트 설정을 유연하게 로드
  async function loadConfig() {
    try {
      const res = await fetch(`${apiHost}/api/config`);
      const data = await res.json();
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

  // 운 점수별 스타일 정보 매핑 함수
  function getLuckThreshold(score: number) {
    let matched = thresholds[0];
    for (const t of thresholds) {
      if (score >= t.minScore) {
        matched = t;
      }
    }
    return matched;
  }

  // 저장된 플레이어 목록 로드
  async function loadPlayers() {
    try {
      const res = await fetch(`${apiHost}/api/players`);
      const data = await res.json();
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
      const res = await fetch(`${apiHost}/api/stats/${playerId}`);
      const data = await res.json();
      if (data.success) {
        activeStats = data.stats;
      } else {
        errorMessage = data.error || "Failed to retrieve stats data";
      }
    } catch (e) {
      errorMessage = "Network connection failed";
    } finally {
      isLoading = false;
    }
  }

  // Kurogame 가챠 결과 원격 스캔 & 트래킹 등록
  async function trackURL() {
    let cleanURL = urlInput.trim();
    if (!cleanURL) {
      errorMessage = "가챠 URL을 바르게 입력해 주세요.";
      return;
    }

    isLoading = true;
    errorMessage = "";
    successMessage = "";

    try {
      const res = await fetch(`${apiHost}/api/track`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({ url: cleanURL }),
      });
      const data = await res.json();
      if (data.success) {
        activePlayerID = data.playerId;
        activeStats = data.stats;
        successMessage = `플레이어 [${data.playerId}] 님의 데이터를 성공적으로 트래킹 및 갱신했습니다!`;
        urlInput = "";
        // 신규 유저 히스토리 갱신
        await loadPlayers();
      } else {
        errorMessage = data.error || "Failed to scan log URL";
      }
    } catch (e) {
      errorMessage = "Network connection failed";
    } finally {
      isLoading = false;
    }
  }

  // 파일 선택 감지 및 로드
  function handleFileChange(event: Event) {
    const input = event.target as HTMLInputElement;
    if (!input.files || input.files.length === 0) return;

    const file = input.files[0];
    uploadFileName = file.name;

    // 파일 이름에서 플레이어 ID 후보군 추출 (예: 20260518143619.json -> 20260518143619)
    const baseName = file.name.replace(/\.[^/.]+$/, "");
    uploadPlayerID = baseName;

    const reader = new FileReader();
    reader.onload = (e) => {
      try {
        const text = e.target?.result as string;
        uploadedJSONData = JSON.parse(text);
        errorMessage = "";
        showUploadModal = true; // 유저가 ID를 확인하거나 수정할 수 있도록 모달 팝업 실행
      } catch (err) {
        errorMessage =
          "올바르지 않은 JSON 파일 형식입니다. 다시 확인해 주세요.";
        uploadedJSONData = null;
        uploadFileName = "";
      }
    };
    reader.readAsText(file);
  }

  // 파일 선택창 트리거 실행
  function triggerFileSelect() {
    if (fileInputRef) {
      fileInputRef.click();
    }
  }

  // JSON 업로드 최종 제출
  async function submitJSONUpload() {
    const cleanID = uploadPlayerID.trim();
    if (!cleanID) {
      alert("플레이어 ID를 입력해 주세요.");
      return;
    }

    if (!uploadedJSONData) {
      alert("업로드할 JSON 데이터가 존재하지 않습니다.");
      return;
    }

    isLoading = true;
    errorMessage = "";
    successMessage = "";
    showUploadModal = false;

    try {
      const res = await fetch(`${apiHost}/api/upload`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({
          playerId: cleanID,
          data: uploadedJSONData,
        }),
      });
      const data = await res.json();
      if (data.success) {
        activePlayerID = data.playerId;
        activeStats = data.stats;
        successMessage = `JSON 파일 [${uploadFileName}]을 오프라인 등록했습니다! [플레이어 ID: ${data.playerId}]`;
        uploadedJSONData = null;
        uploadFileName = "";
        // 신규 유저 히스토리 갱신
        await loadPlayers();
      } else {
        errorMessage = data.error || "Failed to upload JSON log";
      }
    } catch (e) {
      errorMessage = "Network connection failed during JSON upload";
    } finally {
      isLoading = false;
    }
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
  <header
    class="flex flex-col md:flex-row justify-between items-start md:items-center gap-4 mb-8 border-b border-slate-800 pb-6"
  >
    <div>
      <div class="flex items-center gap-2 mb-1">
        <span
          class="inline-flex h-3 w-3 rounded-full bg-indigo-500 animate-pulse"
        ></span>
        <span
          class="text-xs font-semibold tracking-wider text-indigo-400 uppercase"
          >Wuthering Waves</span
        >
      </div>
      <h1
        class="text-3xl md:text-4xl font-extrabold tracking-tight gradient-text"
      >
        튜닝 통계 리포트 대시보드
      </h1>
    </div>
    <div
      class="text-xs text-slate-500 text-left md:text-right bg-slate-900/40 p-3 rounded-lg border border-slate-800"
    >
      <p>제작: Antigravity Tracker</p>
      <p class="mt-1">시간 표시는 로컬 장치 기준입니다.</p>
    </div>
  </header>

  <!-- 컨트롤 판넬 (URL 트래킹 및 유저 목록 전환) -->
  <div class="glass-card p-8 mb-10 relative overflow-hidden">
    <div
      class="absolute top-0 left-0 right-0 h-[3px] bg-gradient-to-r from-indigo-500 via-purple-500 to-pink-500"
    ></div>

    <div class="grid grid-cols-1 lg:grid-cols-3 gap-6">
      <!-- 가챠 URL 입력 영역 -->
      <div class="lg:col-span-2">
        <h2
          class="text-sm font-bold text-slate-300 uppercase tracking-wider mb-2"
        >
          새로운 튜닝 URL 트래킹 / 오프라인 분석
        </h2>
        <div class="flex flex-col sm:flex-row gap-2">
          <input
            type="text"
            placeholder="https://aki-gm-resources..."
            bind:value={urlInput}
            class="flex-1 bg-slate-950/80 border border-slate-800 rounded-xl px-4 py-3 text-sm focus:outline-none focus:border-indigo-500 focus:ring-1 focus:ring-indigo-500 transition-colors"
            disabled={isLoading}
          />
          <div class="flex gap-2">
            <button
              on:click={trackURL}
              class="flex-1 sm:flex-none bg-indigo-600 hover:bg-indigo-500 active:bg-indigo-700 text-white font-bold text-sm px-6 py-3 rounded-xl transition-all shadow-md active:scale-95 disabled:opacity-50 whitespace-nowrap"
              disabled={isLoading}
            >
              {isLoading ? "조회 중..." : "튜닝 트래킹"}
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
              📁 JSON 업로드
            </button>
          </div>
        </div>
      </div>

      <!-- 이전에 스캔한 유저 선택 영역 -->
      <div>
        <h2
          class="text-sm font-bold text-slate-300 uppercase tracking-wider mb-2"
        >
          저장된 플레이어 기록 ({playersList.length})
        </h2>
        {#if playersList.length > 0}
          <div
            class="flex flex-wrap gap-2 max-h-[110px] overflow-y-auto custom-scrollbar p-1"
          >
            {#each playersList as player}
              <button
                on:click={() => selectPlayer(player)}
                class="text-xs px-3 py-1.5 font-bold rounded-lg border transition-all {activePlayerID ===
                player
                  ? 'bg-indigo-600/20 text-indigo-400 border-indigo-500'
                  : 'bg-slate-900/60 text-slate-400 border-slate-800 hover:border-slate-700'}"
                disabled={isLoading}
              >
                👤 {player}
              </button>
            {/each}
          </div>
        {:else}
          <p class="text-xs text-slate-500 italic py-3">
            아직 등록된 플레이어가 없습니다. 위 URL 입력 혹은 JSON 파일 업로드로
            가챠 데이터를 분석 하세요!
          </p>
        {/if}
      </div>
    </div>

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

  <!-- 통계 리포트 뷰어 영역 -->
  {#if activeStats.length > 0}
    <div class="space-y-16">
      {#each activeStats as stat, index}
        {@const luckStyle = getLuckThreshold(stat.luckScore)}
        <div class="glass-card p-8 md:p-10 relative overflow-hidden">
          <!-- 배너 헤더 -->
          <div
            class="flex flex-col sm:flex-row justify-between items-start sm:items-center gap-4 mb-6 border-b border-slate-800/60 pb-4"
          >
            <h3
              class="text-xl md:text-2xl font-bold text-slate-100 flex items-center gap-2"
            >
              <span class="w-2 h-6 bg-indigo-500 rounded"></span>
              {stat.gachaName || "로딩 중"}
            </h3>
            <span
              class="text-xs font-semibold text-slate-500 bg-slate-900 px-3 py-1 rounded-full border border-slate-800"
            >
              Type Code: {stat.gachaType}
            </span>
          </div>

          <!-- 대시보드 2컬럼 레이아웃 그리드 구성 -->
          <div class="grid grid-cols-1 lg:grid-cols-3 gap-8">
            <!-- 좌측 주요 분석 파트 (2/3 차지) -->
            <div class="lg:col-span-2 space-y-6">
              <!-- 핵심 지표 요약 대시보드 -->
              <div class="grid grid-cols-1 sm:grid-cols-3 gap-4">
                <div class="bg-slate-900/40 p-5 rounded-xl border border-slate-800/50">
                  <p class="text-xs text-slate-500 mb-1">총 뽑기 횟수</p>
                  <p class="text-2xl font-extrabold text-slate-100">{stat.totalPulls}</p>
                </div>
                <div class="bg-slate-900/40 p-5 rounded-xl border border-slate-800/50 flex flex-col justify-between">
                  <div>
                    <p class="text-xs text-slate-500 mb-1">현재 5성 천장 스택</p>
                    <p class="text-2xl font-extrabold text-amber-400">
                      {stat.currentPity5}<span class="text-xs text-slate-500 font-normal"> / 80</span>
                    </p>
                  </div>
                  <!-- 비주얼 Pity 게이지 바 -->
                  <div class="mt-2.5">
                    <div class="w-full bg-slate-950/80 rounded-full h-2 overflow-hidden border border-slate-800/80">
                      <div
                        class="bg-gradient-to-r from-indigo-500 via-purple-500 to-amber-500 h-full rounded-full transition-all duration-500"
                        style="width: {Math.min((stat.currentPity5 / 80) * 100, 100)}%"
                      ></div>
                    </div>
                  </div>
                </div>

                <!-- 운 점수 동적 바인딩 -->
                <div
                  class="p-5 rounded-xl border transition-all duration-300 {stat.hasFiveStar
                    ? luckStyle.bgClass
                    : 'bg-slate-900/40 border-slate-800/50'}"
                >
                  <div class="flex items-center gap-1 mb-1">
                    <span class="text-xs text-slate-500">운 점수 (Luck Score)</span>
                    <!-- 가이드 툴팁 -->
                    <div class="group relative cursor-pointer">
                      <svg
                        class="w-3.5 h-3.5 text-slate-500 hover:text-slate-300 transition-colors"
                        fill="none"
                        stroke="currentColor"
                        viewBox="0 0 24 24"
                      >
                        <path
                          stroke-linecap="round"
                          stroke-linejoin="round"
                          stroke-width="2"
                          d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
                        ></path>
                      </svg>
                      <div
                        class="absolute bottom-full left-1/2 -translate-x-1/2 mb-2 w-64 bg-slate-950 border border-slate-800 p-3 rounded-lg text-[11px] leading-relaxed text-slate-300 opacity-0 pointer-events-none group-hover:opacity-100 group-hover:pointer-events-auto transition-all duration-200 shadow-2xl z-30 font-normal"
                      >
                        <p class="font-bold text-indigo-400 mb-1">💡 운 점수 계산 방식</p>
                        <p class="mb-1.5">
                          <code class="bg-slate-900 px-1.5 py-0.5 rounded text-indigo-400 font-mono text-[10px]"
                            >운 점수 = (총 기대 소요 횟수 / 총 실제 소요 횟수) * 100</code
                          >
                        </p>
                        <p class="mb-1">
                          실제 소요된 뽑기 횟수가 기대 평균보다 적을수록 점수가 100% 초과로 상승하며 행운 상태를 나타냅니다.
                        </p>
                        <p class="text-rose-400 border-t border-slate-800/80 pt-1 mt-1">
                          ⚠️ <strong>한정 튜닝 픽업 주기 반영:</strong><br />한정 배너에서 픽뚫(상시 캐릭터)이 발생하면
                          <strong>해당 상시 캐릭터를 뽑는 데 소모된 스택이 다음 픽업 캐릭터 획득 주기(Cycle)에 합산</strong>되어 평가됩니다.
                          픽업 캐릭터를 실제로 획득할 때까지 소모한 물리적인 총 뽑기 횟수를 기대 평균값(55.5)과 정밀 비교합니다.
                        </p>
                        <div class="absolute top-full left-1/2 -translate-x-1/2 -mt-1 border-4 border-transparent border-t-slate-950"></div>
                      </div>
                    </div>
                  </div>
                  <p class="text-2xl font-extrabold">
                    {#if stat.hasFiveStar}
                      <span class={luckStyle.colorClass}>{luckStyle.state}</span>
                      <span class="text-[10px] text-slate-500 font-normal">({stat.luckScore.toFixed(0)}%)</span>
                    {:else}
                      <span class="text-slate-500 italic text-sm">판별 불가 (5성 없음)</span>
                    {/if}
                  </p>
                </div>
              </div>

              <!-- 획득 확률 효율 분석 지표 -->
              <div class="bg-slate-950/50 rounded-xl border border-slate-850 p-5">
                <h4 class="text-xs font-bold text-indigo-400 uppercase tracking-wider mb-3">5★ 획득 효율 분석</h4>
                <div class="grid grid-cols-1 sm:grid-cols-3 gap-4 text-sm">
                  <div class="flex justify-between sm:flex-col sm:justify-start gap-1">
                    <span class="text-slate-500 text-xs">나의 5성 평균 소요</span>
                    <span class="font-bold text-slate-200">
                      {#if stat.hasFiveStar}
                        <span class="text-amber-400 font-extrabold">{stat.avgPulls.toFixed(1)}</span>회
                        <span class="text-xs text-slate-500">(누적 {stat.fiveStars ? stat.fiveStars.length : 0}회)</span>
                      {:else}
                        <span class="text-slate-500 italic">획득 이력 없음</span>
                      {/if}
                    </span>
                  </div>
                  <div class="flex justify-between sm:flex-col sm:justify-start gap-1">
                    <span class="text-slate-500 text-xs">기대 평균 소요 (기준)</span>
                    <span class="font-bold text-slate-200">{stat.expectedPulls}회</span>
                  </div>
                  <div class="flex justify-between sm:flex-col sm:justify-start gap-1">
                    <span class="text-slate-500 text-xs">5★ 획득 비율 / 기본 확률</span>
                    <span class="font-bold text-slate-200">
                      {#if stat.hasFiveStar}
                        <span class={stat.actualRate >= stat.baseRate ? "text-emerald-400 font-extrabold" : "text-rose-400 font-extrabold"}>
                          {stat.actualRate.toFixed(2)}%
                        </span>
                        <span class="text-xs text-slate-500">/ {stat.baseRate.toFixed(1)}%</span>
                      {:else}
                        <span class="text-slate-500">0.00% / {stat.baseRate.toFixed(1)}%</span>
                      {/if}
                    </span>
                  </div>
                </div>
              </div>

              <!-- 5성 히스토리 세션 -->
              <div>
                <h4 class="text-sm font-bold text-slate-400 mb-3 flex items-center gap-2">
                  <span class="w-1.5 h-1.5 bg-amber-500 rounded-full animate-ping"></span>
                  5★ 획득 히스토리 (총 {stat.fiveStars ? stat.fiveStars.length : 0}회)
                </h4>
                {#if stat.fiveStars && stat.fiveStars.length > 0}
                  <div class="overflow-x-auto border border-slate-800/80 rounded-xl bg-slate-950/30">
                    <table class="min-w-full divide-y divide-slate-800">
                      <thead class="bg-slate-900/60">
                        <tr>
                          <th class="px-4 py-3 text-left text-xs font-semibold text-slate-400 uppercase tracking-wider">이름</th>
                          <th class="px-4 py-3 text-left text-xs font-semibold text-slate-400 uppercase tracking-wider">획득 시점 (스택)</th>
                          <th class="px-4 py-3 text-left text-xs font-semibold text-slate-400 uppercase tracking-wider">획득 일시</th>
                          <th class="px-4 py-3 text-left text-xs font-semibold text-slate-400 uppercase tracking-wider">구분</th>
                        </tr>
                      </thead>
                      <tbody class="divide-y divide-slate-800/60 bg-transparent">
                        {#each stat.fiveStars as fs}
                          <tr class="hover:bg-slate-900/20 transition-colors">
                            <td class="px-4 py-3.5 whitespace-nowrap text-sm font-bold text-amber-400 flex items-center gap-1.5">
                              <!-- 별 마크 SVG -->
                              <svg class="w-4 h-4 text-amber-500" fill="currentColor" viewBox="0 0 20 20">
                                <path
                                  d="M9.049 2.927c.3-.921 1.603-.921 1.902 0l1.07 3.292a1 1 0 00.95.69h3.462c.969 0 1.371 1.24.588 1.81l-2.8 2.034a1 1 0 00-.364 1.118l1.07 3.292c.3.921-.755 1.688-1.54 1.118l-2.8-2.034a1 1 0 00-1.175 0l-2.8 2.034c-.784.57-1.838-.197-1.539-1.118l1.07-3.292a1 1 0 00-.364-1.118L2.98 8.72c-.783-.57-.38-1.81.588-1.81h3.461a1 1 0 00.951-.69l1.07-3.292z"
                                ></path>
                              </svg>
                              {fs.name}
                            </td>
                            <td class="px-4 py-3.5 whitespace-nowrap text-sm text-slate-200">
                              <span class="font-extrabold text-amber-500 bg-amber-500/10 px-2 py-0.5 rounded border border-amber-500/20">
                                {fs.pity}스택
                              </span>
                            </td>
                            <td class="px-4 py-3.5 whitespace-nowrap text-xs text-slate-400">{fs.time}</td>
                            <td class="px-4 py-3.5 whitespace-nowrap text-sm">
                              {#if fs.isPickUp}
                                <span class="px-2.5 py-0.5 inline-flex text-xs leading-5 font-semibold rounded-full bg-emerald-500/10 text-emerald-400 border border-emerald-500/20">
                                  픽업 완료
                                </span>
                              {:else}
                                <span class="px-2.5 py-0.5 inline-flex text-xs leading-5 font-semibold rounded-full bg-rose-500/10 text-rose-400 border border-rose-500/20">
                                  픽뚫 (상시)
                                </span>
                              {/if}
                            </td>
                          </tr>
                        {/each}
                      </tbody>
                    </table>
                  </div>
                {:else}
                  <div class="bg-slate-950/20 border border-dashed border-slate-800 rounded-xl p-6 text-center text-slate-500 text-sm">
                    아직 5성 캐릭터/무기 획득 이력이 없습니다.
                  </div>
                {/if}
              </div>
            </div>

            <!-- 우측 전체 히스토리 리스트 파트 (1/3 차지) -->
            <div class="flex flex-col h-[650px] bg-slate-950/40 p-5 rounded-2xl border border-slate-900/60">
              <h4 class="text-sm font-bold text-slate-300 mb-4 flex items-center gap-2 pb-2.5 border-b border-slate-900/60">
                <svg class="w-4 h-4 text-indigo-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 6h16M4 12h16M4 18h16"></path>
                </svg>
                전체 튜닝 기록 (최신순 - 총 {stat.records ? stat.records.length : 0}회)
              </h4>
              
              <div class="flex-1 overflow-y-auto custom-scrollbar pr-1">
                {#if stat.records && stat.records.length > 0}
                  <table class="min-w-full divide-y divide-slate-900">
                    <thead class="bg-slate-950/80 sticky top-0 z-10">
                      <tr>
                        <th class="px-3 py-2 text-left text-[11px] font-semibold text-slate-500 uppercase tracking-wider">이름</th>
                        <th class="px-3 py-2 text-left text-[11px] font-semibold text-slate-500 uppercase tracking-wider">등급</th>
                        <th class="px-3 py-2 text-left text-[11px] font-semibold text-slate-500 uppercase tracking-wider">유형</th>
                      </tr>
                    </thead>
                    <tbody class="divide-y divide-slate-900/50 bg-transparent">
                      {#each stat.records as r}
                        <tr
                          class="hover:bg-slate-900/30 transition-colors {r.qualityLevel === 5 ? 'bg-amber-500/5' : r.qualityLevel === 4 ? 'bg-purple-500/5' : ''}"
                        >
                          <td class="px-3 py-2.5 whitespace-nowrap text-xs {r.qualityLevel === 5 ? 'font-bold text-amber-400' : r.qualityLevel === 4 ? 'font-semibold text-purple-300' : 'text-slate-300'}">
                            {r.name}
                          </td>
                          <td class="px-3 py-2.5 whitespace-nowrap text-[10px]">
                            {#if r.qualityLevel === 5}
                              <span class="px-1.5 py-0.5 inline-flex leading-3 font-bold rounded bg-amber-500/10 text-amber-400 border border-amber-500/20">5★</span>
                            {:else if r.qualityLevel === 4}
                              <span class="px-1.5 py-0.5 inline-flex leading-3 font-semibold rounded bg-purple-500/10 text-purple-300 border border-purple-500/20">4★</span>
                            {:else}
                              <span class="px-1.5 py-0.5 inline-flex leading-3 rounded bg-slate-800 text-slate-400 border border-slate-700/50">3★</span>
                            {/if}
                          </td>
                          <td class="px-3 py-2.5 whitespace-nowrap text-[10px] text-slate-500">
                            {r.resourceType}
                          </td>
                        </tr>
                      {/each}
                    </tbody>
                  </table>
                {:else}
                  <div class="p-6 text-center text-slate-500 text-xs italic">
                    획득 기록이 없습니다.
                  </div>
                {/if}
              </div>
            </div>
          </div>
        </div>
      {/each}
    </div>
  {:else if !isLoading}
    <div class="glass-card p-12 text-center text-slate-500 border-dashed">
      <span class="text-4xl block mb-3">🔮</span>
      <p class="text-base font-bold text-slate-300 mb-1">
        조회된 통계 데이터가 없습니다.
      </p>
      <p class="text-xs">
        상단의 URL 입력창 혹은 JSON 파일 업로드 기능을 통해 분석을 진행하세요.
      </p>
    </div>
  {/if}
</div>

<!-- JSON 업로드용 모달 오버레이 -->
{#if showUploadModal}
  <div
    class="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/80 backdrop-blur-sm px-4"
  >
    <div
      class="glass-card max-w-md w-full p-6 border border-slate-800 shadow-2xl relative"
    >
      <h3 class="text-lg font-bold text-slate-100 mb-2 flex items-center gap-2">
        <span>📁</span> 오프라인 JSON 데이터 등록
      </h3>
      <p class="text-xs text-slate-400 mb-5 leading-relaxed">
        선택한 파일: <strong class="text-indigo-400 font-mono"
          >{uploadFileName}</strong
        ><br />
        이 파일 데이터를 저장할 플레이어 ID를 설정하세요. 기존에 존재하는 플레이어
        ID를 입력하면 해당 플레이어의 기록이 이 파일 내용으로 완전히 덮어씌워집니다.
      </p>

      <div class="mb-5">
        <label
          for="playerIdInput"
          class="block text-xs font-semibold uppercase tracking-wider text-slate-400 mb-1.5"
          >플레이어 ID</label
        >
        <input
          id="playerIdInput"
          type="text"
          placeholder="예: player_12345"
          bind:value={uploadPlayerID}
          class="w-full bg-slate-950/90 border border-slate-800 rounded-xl px-4 py-3 text-sm focus:outline-none focus:border-indigo-500 focus:ring-1 focus:ring-indigo-500 transition-colors text-slate-200"
        />
      </div>

      <div class="flex justify-end gap-2">
        <button
          on:click={() => {
            showUploadModal = false;
            uploadedJSONData = null;
            uploadFileName = "";
          }}
          class="px-4 py-2.5 rounded-lg text-slate-400 hover:text-slate-200 text-xs font-semibold bg-slate-900 border border-slate-800 hover:border-slate-700 transition-colors"
        >
          취소
        </button>
        <button
          on:click={submitJSONUpload}
          class="px-5 py-2.5 rounded-lg bg-indigo-600 hover:bg-indigo-500 active:bg-indigo-700 text-white text-xs font-bold transition-all"
        >
          등록 및 분석 시작
        </button>
      </div>
    </div>
  </div>
{/if}
