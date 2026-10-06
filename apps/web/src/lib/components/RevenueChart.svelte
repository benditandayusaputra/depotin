<script lang="ts">
  import { formatShortDay } from '$lib/utils/date';
  import { formatRupiah } from '$lib/utils/rupiah';

  type Point = { day: string; revenue: number; gallons: number };

  let { daily }: { daily: Point[] } = $props();

  const WIDTH = 640;
  const HEIGHT = 220;
  const PAD = { top: 16, right: 8, bottom: 28, left: 8 };
  const GAP = 2;

  const plotWidth = WIDTH - PAD.left - PAD.right;
  const plotHeight = HEIGHT - PAD.top - PAD.bottom;
  const baseline = PAD.top + plotHeight;

  const max = $derived(Math.max(1, ...daily.map((point) => point.revenue)));
  const peakIndex = $derived(daily.findIndex((point) => point.revenue === max));
  const slot = $derived(daily.length ? plotWidth / daily.length : plotWidth);
  const barWidth = $derived(Math.max(2, slot - GAP));
  const labelEvery = $derived(Math.max(1, Math.ceil(daily.length / 6)));
  const gridLines = $derived([0.5, 1].map((ratio) => baseline - plotHeight * ratio));
  const LABEL_MARGIN = 60;

  function peakAnchor(x: number): 'start' | 'middle' | 'end' {
    if (x < PAD.left + LABEL_MARGIN) return 'start';
    if (x > WIDTH - PAD.right - LABEL_MARGIN) return 'end';
    return 'middle';
  }
</script>

<figure class="flex flex-col gap-2">
  <svg
    viewBox="0 0 {WIDTH} {HEIGHT}"
    class="w-full"
    role="img"
    aria-label="Grafik omzet harian, puncak {formatRupiah(max)}"
  >
    {#each gridLines as y (y)}
      <line
        x1={PAD.left}
        x2={WIDTH - PAD.right}
        y1={y}
        y2={y}
        class="stroke-border"
        stroke-width="1"
      />
    {/each}
    <line
      x1={PAD.left}
      x2={WIDTH - PAD.right}
      y1={baseline}
      y2={baseline}
      class="stroke-neutral-400"
      stroke-width="1"
    />
    {#each daily as point, index (point.day)}
      {@const height = (point.revenue / max) * plotHeight}
      {@const x = PAD.left + index * slot + GAP / 2}
      <g>
        <title
          >{formatShortDay(point.day)}: {formatRupiah(point.revenue)}, {point.gallons} galon</title
        >
        <rect
          x={x - GAP / 2}
          y={PAD.top}
          width={slot}
          height={plotHeight}
          class="fill-transparent hover:fill-accent-100/60 dark:hover:fill-accent-900/40"
        />
        {#if point.revenue > 0}
          <path
            d="M{x},{baseline} v{-Math.max(height - 4, 0)} q0,-4 4,-4 h{barWidth -
              8} q4,0 4,4 v{Math.max(height - 4, 0)} z"
            class="fill-accent-600 dark:fill-accent-400"
          />
        {/if}
        {#if index === peakIndex}
          <text
            x={x + barWidth / 2}
            y={baseline - height - 6}
            text-anchor={peakAnchor(x + barWidth / 2)}
            class="fill-text text-[12px] font-semibold"
          >
            {formatRupiah(point.revenue)}
          </text>
        {/if}
        {#if index % labelEvery === 0}
          <text
            x={x + barWidth / 2}
            y={HEIGHT - 8}
            text-anchor="middle"
            class="fill-muted text-[11px]"
          >
            {formatShortDay(point.day)}
          </text>
        {/if}
      </g>
    {/each}
  </svg>
  <details>
    <summary class="min-h-11 cursor-pointer font-medium text-accent-700 dark:text-accent-300">
      Lihat tabel harian
    </summary>
    <table class="mt-2 w-full text-sm">
      <thead>
        <tr class="text-left text-muted">
          <th class="py-1 font-medium">Tanggal</th>
          <th class="py-1 text-right font-medium">Omzet</th>
          <th class="py-1 text-right font-medium">Galon</th>
        </tr>
      </thead>
      <tbody>
        {#each daily as point (point.day)}
          <tr class="border-t border-border">
            <td class="py-1">{formatShortDay(point.day)}</td>
            <td class="py-1 text-right">{formatRupiah(point.revenue)}</td>
            <td class="py-1 text-right">{point.gallons}</td>
          </tr>
        {/each}
      </tbody>
    </table>
  </details>
</figure>
