<script lang="ts">
  import type { FiveStarRecord } from "../types";
  import { t } from "../i18n";

  // Props 정의
  export let fiveStars: FiveStarRecord[] | null;
</script>

{#if fiveStars && fiveStars.length > 0}
  <div class="overflow-x-auto border border-slate-800/80 rounded-xl bg-slate-950/30">
    <table class="min-w-full divide-y divide-slate-800">
      <thead class="bg-slate-900/60">
        <tr>
          <th class="px-4 py-3 text-left text-xs font-semibold text-slate-400 uppercase tracking-wider">
            {$t("table.name")}
          </th>
          <th class="px-4 py-3 text-left text-xs font-semibold text-slate-400 uppercase tracking-wider">
            {$t("table.pity")}
          </th>
          <th class="px-4 py-3 text-left text-xs font-semibold text-slate-400 uppercase tracking-wider">
            {$t("table.time")}
          </th>
          <th class="px-4 py-3 text-left text-xs font-semibold text-slate-400 uppercase tracking-wider">
            {$t("table.category")}
          </th>
        </tr>
      </thead>
      <tbody class="divide-y divide-slate-800/60 bg-transparent">
        {#each fiveStars as fs}
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
                {$t("table.pity_suffix", { count: fs.pity })}
              </span>
            </td>
            <td class="px-4 py-3.5 whitespace-nowrap text-xs text-slate-400">{fs.time}</td>
            <td class="px-4 py-3.5 whitespace-nowrap text-sm">
              {#if fs.isPickUp}
                <span class="px-2.5 py-0.5 inline-flex text-xs leading-5 font-semibold rounded-full bg-emerald-500/10 text-emerald-400 border border-emerald-500/20">
                  {$t("table.pickup_success")}
                </span>
              {:else}
                <span class="px-2.5 py-0.5 inline-flex text-xs leading-5 font-semibold rounded-full bg-rose-500/10 text-rose-400 border border-rose-500/20">
                  {$t("table.pickup_fail")}
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
    {$t("report.history_empty")}
  </div>
{/if}
