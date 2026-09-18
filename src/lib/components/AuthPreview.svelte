<script lang="ts">
  import logo from '#src/assets/garage-logo.svg';
</script>

<!-- Preserve the original author's split-screen sign-in design. -->
<aside
  class="relative hidden flex-col items-center justify-center overflow-hidden bg-gradient-to-br from-violet-200 via-white to-slate-100 px-12 py-10 md:flex"
>
  <div
    aria-hidden="true"
    class="w-full max-w-md overflow-hidden rounded-xl border border-slate-800 bg-[#0b1220] shadow-2xl shadow-slate-900/30"
  >
    <div class="flex items-center gap-1.5 border-b border-white/10 px-3 py-2.5">
      <span class="h-2.5 w-2.5 rounded-full bg-rose-400"></span>
      <span class="h-2.5 w-2.5 rounded-full bg-amber-400"></span>
      <span class="h-2.5 w-2.5 rounded-full bg-emerald-400"></span>
    </div>
    <div class="flex">
      <div class="w-1/3 space-y-2 border-r border-white/10 p-3">
        <div class="mb-3 flex items-center gap-1.5">
          <img src={logo} alt="" class="h-5 w-5" />
          <div class="h-2 w-10 rounded bg-white/20"></div>
        </div>
        <div class="h-6 rounded bg-cyan-400/80"></div>
        {#each [1, 2, 3] as row (row)}<div
            class="h-6 rounded bg-white/5"
          ></div>{/each}
      </div>
      <div class="flex-1 p-3">
        <div class="mb-3 h-2 w-16 rounded bg-white/20"></div>
        <div class="grid grid-cols-2 gap-2">
          {#each [1, 2, 3, 4] as card (card)}<div
              class="rounded-lg border border-white/10 bg-white/5 p-2.5"
            >
              <div
                class="mb-1.5 h-3 w-8 rounded {card === 1
                  ? 'bg-emerald-400/80'
                  : 'bg-white/30'}"
              ></div>
              <div class="h-1.5 w-12 rounded bg-white/10"></div>
            </div>{/each}
        </div>
      </div>
    </div>
  </div>
  <div class="mt-10 max-w-md text-center">
    <h2 class="text-2xl font-bold text-slate-900">
      Manage your object storage with ease
    </h2>
    <p class="mt-3 text-sm text-slate-500">
      Garage Web UI — browse buckets, manage access keys, and monitor cluster
      health, all in one place.
    </p>
  </div>
</aside>
