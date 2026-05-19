<script lang="ts">
  import { t } from "../i18n";

  // Props 정의
  export let show: boolean;
  export let uploadFileName: string;
  export let uploadPlayerID: string;

  // 콜백 함수 정의
  export let onCancel: () => void;
  export let onSubmit: () => void;
</script>

{#if show}
  <!-- JSON 업로드용 모달 오버레이 -->
  <div
    class="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/80 backdrop-blur-sm px-4"
  >
    <div
      class="glass-card max-w-md w-full p-6 border border-slate-800 shadow-2xl relative"
    >
      <h3 class="text-lg font-bold text-slate-100 mb-2 flex items-center gap-2">
        <span>📁</span> {$t("upload.modal_title")}
      </h3>
      <p class="text-xs text-slate-400 mb-5 leading-relaxed">
        {$t("upload.selected_file", { fileName: uploadFileName })}
        <br />
        {$t("upload.description")}
      </p>

      <div class="mb-5">
        <label
          for="playerIdInput"
          class="block text-xs font-semibold uppercase tracking-wider text-slate-400 mb-1.5"
        >
          {$t("upload.player_id_label")}
        </label>
        <input
          id="playerIdInput"
          type="text"
          placeholder={$t("upload.player_id_placeholder")}
          bind:value={uploadPlayerID}
          class="w-full bg-slate-950/90 border border-slate-800 rounded-xl px-4 py-3 text-sm focus:outline-none focus:border-indigo-500 focus:ring-1 focus:ring-indigo-500 transition-colors text-slate-200"
        />
      </div>

      <div class="flex justify-end gap-2">
        <button
          on:click={onCancel}
          class="px-4 py-2.5 rounded-lg text-slate-400 hover:text-slate-200 text-xs font-semibold bg-slate-900 border border-slate-800 hover:border-slate-700 transition-colors"
        >
          {$t("upload.cancel")}
        </button>
        <button
          on:click={onSubmit}
          class="px-5 py-2.5 rounded-lg bg-indigo-600 hover:bg-indigo-500 active:bg-indigo-700 text-white text-xs font-bold transition-all"
        >
          {$t("upload.submit")}
        </button>
      </div>
    </div>
  </div>
{/if}
