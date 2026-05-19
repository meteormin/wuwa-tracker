<script lang="ts">
  import type { Record } from "../types";
  import { t } from "../i18n";

  // Props 정의
  export let records: Record[] | null;
</script>

<!-- 우측 전체 히스토리 리스트 파트 (1/3 차지) -->
<div class="flex flex-col h-[650px] bg-slate-950/40 p-5 rounded-2xl border border-slate-900/60">
  <h4 class="text-sm font-bold text-slate-300 mb-4 flex items-center gap-2 pb-2.5 border-b border-slate-900/60">
    <svg class="w-4 h-4 text-indigo-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 6h16M4 12h16M4 18h16"></path>
    </svg>
    {$t("full_history.title", { count: records ? records.length : 0 })}
  </h4>

  <div class="flex-1 overflow-y-auto custom-scrollbar pr-1">
    {#if records && records.length > 0}
      <table class="min-w-full divide-y divide-slate-900">
        <thead class="bg-slate-950/80 sticky top-0 z-10">
          <tr>
            <th class="px-3 py-2 text-left text-[11px] font-semibold text-slate-500 uppercase tracking-wider">
              {$t("table.name")}
            </th>
            <th class="px-3 py-2 text-left text-[11px] font-semibold text-slate-500 uppercase tracking-wider">
              {$t("table.rarity")}
            </th>
            <th class="px-3 py-2 text-left text-[11px] font-semibold text-slate-500 uppercase tracking-wider">
              {$t("table.type")}
            </th>
          </tr>
        </thead>
        <tbody class="divide-y divide-slate-900/50 bg-transparent">
          {#each records as r}
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
        {$t("full_history.empty")}
      </div>
    {/if}
  </div>
</div>
