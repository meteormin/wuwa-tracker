<script lang="ts">
  import { locale, t } from "../i18n";

  // 로케일 언어 변경 함수
  function changeLocale(newLocale: "ko" | "en") {
    locale.set(newLocale);
  }
</script>

<!-- 대시보드 상단 헤더 컴포넌트 -->
<header
  class="flex flex-col md:flex-row justify-between items-start md:items-center gap-6 mb-8 border-b border-slate-800 pb-6"
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
      {$t('header.title')}
    </h1>
  </div>

  <div class="flex flex-col sm:flex-row items-start sm:items-center gap-4">
    <!-- 언어 선택 토글 스위치 -->
    <div class="flex items-center bg-slate-950/60 p-1 rounded-lg border border-slate-800/80">
      <button
        class="px-2.5 py-1 text-[11px] font-bold rounded transition-all duration-200 {$locale === 'ko' ? 'bg-indigo-600/35 text-indigo-400 border border-indigo-500/30' : 'text-slate-500 border border-transparent hover:text-slate-300'}"
        on:click={() => changeLocale('ko')}
      >
        한국어
      </button>
      <button
        class="px-2.5 py-1 text-[11px] font-bold rounded transition-all duration-200 {$locale === 'en' ? 'bg-indigo-600/35 text-indigo-400 border border-indigo-500/30' : 'text-slate-500 border border-transparent hover:text-slate-300'}"
        on:click={() => changeLocale('en')}
      >
        English
      </button>
    </div>

    <!-- 정보 표시 패널 -->
    <div
      class="text-[11px] text-slate-500 bg-slate-900/40 p-2.5 rounded-lg border border-slate-800"
    >
      <p>{$t('header.author')}</p>
      <p class="mt-0.5">{$t('header.timezone_warning')}</p>
    </div>
  </div>
</header>
