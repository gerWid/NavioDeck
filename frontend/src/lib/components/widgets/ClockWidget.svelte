<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import type { Widget } from '$lib/types';

	let { widget }: { widget: Widget } = $props();

	const clockStyle  = $derived((widget.config?.style  as string) || 'digital');
	const format      = $derived((widget.config?.format  as string) || '24h');
	const showDate    = $derived(widget.config?.show_date !== false);
	const showSeconds = $derived(widget.config?.show_seconds !== false);
	const timezone    = $derived((widget.config?.timezone as string) || 'Europe/Berlin');

	let now = $state(new Date());
	let interval: ReturnType<typeof setInterval>;

	onMount(() => { interval = setInterval(() => (now = new Date()), 1000); });
	onDestroy(() => clearInterval(interval));

	// Timezone-aware time parts for analog hands
	const tzParts = $derived(() => {
		// Create a Date string in the target timezone and parse it back
		const str = now.toLocaleString('en-US', {
			hour: 'numeric', minute: 'numeric', second: 'numeric',
			hour12: false, timeZone: timezone,
		});
		// str format: "HH:MM:SS" or "H:MM:SS"
		const [h, m, s] = str.split(':').map(Number);
		return { h: h % 24, m, s };
	});

	const hourAngle   = $derived(() => { const { h, m, s } = tzParts(); return (h % 12 + m / 60 + s / 3600) * 30; });
	const minuteAngle = $derived(() => { const { m, s }    = tzParts(); return (m + s / 60) * 6; });
	const secondAngle = $derived(() => tzParts().s * 6);

	// Tick positions (60 ticks, every 6°)
	const ticks = Array.from({ length: 60 }, (_, i) => i);

	const timeStr = $derived(() => {
		return new Intl.DateTimeFormat('de-DE', {
			hour: '2-digit', minute: '2-digit', second: '2-digit',
			hour12: format === '12h', timeZone: timezone,
		}).format(now);
	});

	const dateStr = $derived(() => {
		return new Intl.DateTimeFormat('de-DE', {
			weekday: 'long', day: 'numeric', month: 'long', year: 'numeric',
			timeZone: timezone,
		}).format(now);
	});
</script>

{#if clockStyle === 'analog'}
	<div class="clock analog-clock">
		<svg viewBox="0 0 200 200" class="face" aria-label="Analoge Uhr">
			<!-- Outer ring -->
			<circle cx="100" cy="100" r="90" class="ring" />

			<!-- Tick marks -->
			{#each ticks as i}
				{#if i % 5 === 0}
					<line x1="100" y1="13" x2="100" y2="26" class="tick-h"
						transform="rotate({i * 6} 100 100)" />
				{:else}
					<line x1="100" y1="17" x2="100" y2="24" class="tick-m"
						transform="rotate({i * 6} 100 100)" />
				{/if}
			{/each}

			<!-- Hour hand -->
			<line x1="100" y1="113" x2="100" y2="52"
				stroke-width="5" stroke-linecap="round" class="hand hour"
				transform="rotate({hourAngle()} 100 100)" />

			<!-- Minute hand -->
			<line x1="100" y1="116" x2="100" y2="31"
				stroke-width="3" stroke-linecap="round" class="hand minute"
				transform="rotate({minuteAngle()} 100 100)" />

			<!-- Second hand -->
			{#if showSeconds}
				<line x1="100" y1="122" x2="100" y2="25"
					stroke-width="1.5" stroke-linecap="round" class="hand second"
					transform="rotate({secondAngle()} 100 100)" />
			{/if}

			<!-- Center dot -->
			<circle cx="100" cy="100" r="5" class="center-dot" />
			{#if showSeconds}
				<circle cx="100" cy="100" r="2.5" class="center-inner" />
			{/if}
		</svg>

		{#if showDate}
			<div class="date">{dateStr()}</div>
		{/if}
	</div>

{:else}
	<div class="clock digital-clock">
		<div class="time">{timeStr()}</div>
		{#if showDate}
			<div class="date">{dateStr()}</div>
		{/if}
	</div>
{/if}

<style>
	.clock {
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		height: 100%;
		gap: 8px;
		text-align: center;
		padding: 8px;
	}

	/* ── Digital ─────────────────────────────────── */
	.time {
		font-size: clamp(22px, 4vw, 42px);
		font-weight: 700;
		letter-spacing: -0.02em;
		color: var(--text);
		font-variant-numeric: tabular-nums;
	}

	/* ── Analog ──────────────────────────────────── */
	.analog-clock { gap: 6px; }

	.face {
		width: 100%;
		/* keep square, fill available height minus date line */
		max-height: calc(100% - 24px);
		overflow: visible;
	}

	.ring {
		fill: none;
		stroke: var(--primary);
		stroke-width: 1.5;
		opacity: 0.25;
	}

	.tick-h {
		stroke: var(--text);
		stroke-width: 2.5;
		stroke-linecap: round;
		opacity: 0.75;
	}

	.tick-m {
		stroke: var(--text-muted);
		stroke-width: 1;
		stroke-linecap: round;
		opacity: 0.45;
	}

	.hand {
		stroke: var(--text);
	}

	.hand.second {
		stroke: var(--accent);
	}

	.center-dot {
		fill: var(--primary);
	}

	.center-inner {
		fill: var(--accent);
	}

	/* ── Shared ──────────────────────────────────── */
	.date {
		font-size: clamp(10px, 1.5vw, 13px);
		color: var(--text-muted);
		font-weight: 400;
	}
</style>
