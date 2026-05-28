<script lang="ts">
  import type { Stats, LuckScoreState, LuckScoreThreshold } from "../types";
  import { t } from "../i18n";
  import FiveStarHistory from "./FiveStarHistory.svelte";
  import FullHistory from "./FullHistory.svelte";

  type LuckScoreStateClasses = {
    colorClass: string;
    bgClass: string;
  };

  // Props 정의
  export let stat: Stats;
  export let thresholds: LuckScoreThreshold[];

  // 운 점수에 따른 스타일 동적 계산
  $: luckStyle = getLuckThreshold(stat.luckScore);
  $: luckStyleClasses = getLuckStateClasses(luckStyle.state);

  const DEFAULT_LUCK_STYLE: LuckScoreThreshold = {
    minScore: 0.0,
    state: "normal",
  };

  const LUCK_SCORE_STATE_CLASSES: Record<LuckScoreState, LuckScoreStateClasses> = {
    worst: {
      colorClass: "text-rose-500 font-extrabold",
      bgClass: "bg-rose-500/10 border-rose-500/30",
    },
    bad: {
      colorClass: "text-rose-300",
      bgClass: "bg-rose-500/5 border-rose-500/10",
    },
    normal: {
      colorClass: "text-slate-300",
      bgClass: "bg-slate-900/40 border-slate-800/50",
    },
    good: {
      colorClass: "text-emerald-400",
      bgClass: "bg-emerald-500/5 border-emerald-500/20",
    },
    best: {
      colorClass: "text-emerald-400",
      bgClass: "bg-emerald-500/10 border-emerald-500/30",
    },
  };

  // 운 점수 임계치 정보 매칭 함수
  function getLuckThreshold(score: number) {
    if (!thresholds || thresholds.length === 0) {
      return DEFAULT_LUCK_STYLE;
    }
    let matched = thresholds[0];
    for (const t of thresholds) {
      if (score >= t.minScore) {
        matched = t;
      }
    }
    return matched;
  }

  function getLuckStateClasses(state: LuckScoreState) {
    return LUCK_SCORE_STATE_CLASSES[state] || LUCK_SCORE_STATE_CLASSES.normal;
  }

  function getLuckStateMessage(state: LuckScoreState) {
    return $t(`report.luck_score_state.${state}` as any);
  }
</script>

<div class="glass-card p-8 md:p-10 relative overflow-hidden">
  <!-- 배너 헤더 -->
  <div
    class="flex flex-col sm:flex-row justify-between items-start sm:items-center gap-4 mb-6 border-b border-slate-800/60 pb-4"
  >
    <h3
      class="text-xl md:text-2xl font-bold text-slate-100 flex items-center gap-2"
    >
      <span class="w-2 h-6 bg-blue-500 rounded"></span>
      {stat.gachaName || $t("report.loading")}
    </h3>
    <span
      class="text-xs font-semibold text-slate-500 bg-slate-900 px-3 py-1 rounded-full border border-slate-800"
    >
      {$t("report.type_code", { code: stat.gachaType })}
    </span>
  </div>

  <!-- 대시보드 2컬럼 레이아웃 그리드 구성 -->
  <div class="grid grid-cols-1 lg:grid-cols-3 gap-8">
    <!-- 좌측 주요 분석 파트 (2/3 차지) -->
    <div class="lg:col-span-2 space-y-6">
      <!-- 핵심 지표 요약 대시보드 -->
      <div class="grid grid-cols-1 sm:grid-cols-3 gap-4">
        <div class="bg-slate-900/40 p-5 rounded-xl border border-slate-800/50">
          <p class="text-xs text-slate-500 mb-1">{$t("report.total_pulls")}</p>
          <p class="text-2xl font-extrabold text-slate-100">
            {stat.totalPulls}
          </p>
        </div>
        <div
          class="bg-slate-900/40 p-5 rounded-xl border border-slate-800/50 flex flex-col justify-between"
        >
          <div>
            <p class="text-xs text-slate-500 mb-1">{$t("report.current_pity")}</p>
            <p class="text-2xl font-extrabold text-amber-400">
              {stat.currentPity5}<span
                class="text-xs text-slate-500 font-normal"
              >
                {$t("report.pity_suffix")}</span
              >
            </p>
          </div>
          <!-- 비주얼 Pity 게이지 바 -->
          <div class="mt-2.5">
            <div
              class="w-full bg-slate-950/80 rounded-full h-2 overflow-hidden border border-slate-800/80"
            >
              <div
                class="bg-amber-500 h-full rounded-full transition-all duration-500"
                style="width: {Math.min((stat.currentPity5 / 80) * 100, 100)}%"
              ></div>
            </div>
          </div>
        </div>

        <!-- 운 점수 동적 바인딩 -->
        <div
          class="p-5 rounded-xl border transition-all duration-300 {stat.hasFiveStar
            ? luckStyleClasses.bgClass
            : 'bg-slate-900/40 border-slate-800/50'}"
        >
          <div class="flex items-center gap-1 mb-1">
            <span class="text-xs text-slate-500">{$t("report.luck_score")}</span>
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
                <p class="font-bold text-blue-400 mb-1">
                  {$t("report.luck_score_formula_title")}
                </p>
                <p class="mb-1.5">
                  <code
                    class="bg-slate-900 px-1.5 py-0.5 rounded text-blue-400 font-mono text-[10px]"
                    >{$t("report.luck_score_formula")}</code
                  >
                </p>
                <p class="mb-1">
                  {$t("report.luck_score_description")}
                </p>
                <p class="text-rose-400 border-t border-slate-800/80 pt-1 mt-1 whitespace-pre-line">
                  {$t("report.luck_score_pity_warn")}
                </p>
                <div
                  class="absolute top-full left-1/2 -translate-x-1/2 -mt-1 border-4 border-transparent border-t-slate-950"
                ></div>
              </div>
            </div>
          </div>
          <p class="text-2xl font-extrabold">
            {#if stat.hasFiveStar}
              <span class={luckStyleClasses.colorClass}>{getLuckStateMessage(luckStyle.state)}</span>
              <span class="text-[10px] text-slate-500 font-normal"
                >({stat.luckScore.toFixed(0)}%)</span
              >
            {:else}
              <span class="text-slate-500 italic text-sm"
                >{$t("report.luck_score_unknown")}</span
              >
            {/if}
          </p>
        </div>
      </div>

      <!-- 획득 확률 효율 분석 지표 -->
      <div class="bg-slate-950/50 rounded-xl border border-slate-850 p-5">
        <h4
          class="text-xs font-bold text-blue-400 uppercase tracking-wider mb-3"
        >
          {$t("report.efficiency_analysis")}
        </h4>
        <div class="grid grid-cols-1 sm:grid-cols-3 gap-4 text-sm">
          <div class="flex justify-between sm:flex-col sm:justify-start gap-1">
            <span class="text-slate-500 text-xs">{$t("report.my_avg_pulls")}</span>
            <span class="font-bold text-slate-200">
              {#if stat.hasFiveStar}
                <span class="text-amber-400 font-extrabold"
                  >{stat.avgPulls.toFixed(1)}</span
                >{$t("report.avg_pulls_suffix")}
                <span class="text-xs text-slate-500"
                  >{$t("report.avg_pulls_accumulated", { count: stat.fiveStars ? stat.fiveStars.length : 0 })}</span
                >
              {:else}
                <span class="text-slate-500 italic">{$t("report.no_avg_pulls")}</span>
              {/if}
            </span>
          </div>
          <div class="flex justify-between sm:flex-col sm:justify-start gap-1">
            <span class="text-slate-500 text-xs">{$t("report.expected_avg_pulls")}</span>
            <span class="font-bold text-slate-200">{stat.expectedPulls}{$t("report.avg_pulls_suffix")}</span>
          </div>
          <div class="flex justify-between sm:flex-col sm:justify-start gap-1">
            <span class="text-slate-500 text-xs">{$t("report.actual_rate_vs_base")}</span>
            <span class="font-bold text-slate-200">
              {#if stat.hasFiveStar}
                <span
                  class={stat.actualRate >= stat.baseRate
                    ? "text-emerald-400 font-extrabold"
                    : "text-rose-400 font-extrabold"}
                >
                  {stat.actualRate.toFixed(2)}%
                </span>
                <span class="text-xs text-slate-500"
                  >/ {stat.baseRate.toFixed(1)}%</span
                >
              {:else}
                <span class="text-slate-500"
                  >0.00% / {stat.baseRate.toFixed(1)}%</span
                >
              {/if}
            </span>
          </div>
        </div>
      </div>

      <!-- 5성 히스토리 세션 -->
      <div>
        <h4
          class="text-sm font-bold text-slate-400 mb-3 flex items-center gap-2"
        >
          <span class="w-1.5 h-1.5 bg-amber-500 rounded-full"
          ></span>
          {$t("report.history_title", { count: stat.fiveStars ? stat.fiveStars.length : 0 })}
        </h4>
        <FiveStarHistory fiveStars={stat.fiveStars} />
      </div>
    </div>

    <!-- 우측 전체 히스토리 리스트 파트 (1/3 차지) -->
    <FullHistory records={stat.records} />
  </div>
</div>
