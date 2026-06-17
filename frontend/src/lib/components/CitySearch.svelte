<script lang="ts">
	interface GeoResult {
		id: number;
		name: string;
		admin1?: string;
		country?: string;
		country_code?: string;
	}

	let {
		city = $bindable(),
	}: {
		city: string;
	} = $props();

	let query = $state(city ?? '');
	let results = $state<GeoResult[]>([]);
	let loading = $state(false);
	let open = $state(false);
	let debounceTimer: ReturnType<typeof setTimeout>;
	let wrapper: HTMLElement;

	function onInput(e: Event) {
		query = (e.target as HTMLInputElement).value;
		city = query;
		clearTimeout(debounceTimer);
		if (query.trim().length < 2) {
			results = [];
			open = false;
			return;
		}
		debounceTimer = setTimeout(search, 380);
	}

	async function search() {
		loading = true;
		try {
			const resp = await fetch(
				`https://geocoding-api.open-meteo.com/v1/search?name=${encodeURIComponent(query)}&count=6&language=de`
			);
			const data = await resp.json();
			results = data.results ?? [];
			open = results.length > 0;
		} catch {
			results = [];
		} finally {
			loading = false;
		}
	}

	function select(r: GeoResult) {
		city = r.name;
		query = r.name;
		open = false;
		results = [];
	}

	function onKeydown(e: KeyboardEvent) {
		if (e.key === 'Escape') open = false;
	}

	function onBlur() {
		// Small delay so click on result registers before closing
		setTimeout(() => { open = false; }, 150);
	}

	function label(r: GeoResult) {
		const parts = [r.admin1, r.country].filter(Boolean);
		return parts.length ? `${r.name}, ${parts.join(', ')}` : r.name;
	}
</script>

<div class="city-search" bind:this={wrapper}>
	<div class="input-wrap">
		<input
			type="text"
			value={query}
			oninput={onInput}
			onkeydown={onKeydown}
			onblur={onBlur}
			placeholder="Stadt suchen…"
			autocomplete="off"
		/>
		{#if loading}
			<span class="spinner" aria-hidden="true">⟳</span>
		{/if}
	</div>

	{#if open && results.length > 0}
		<div class="dropdown" role="listbox">
			{#each results as r (r.id)}
				<button
					class="result"
					role="option"
					aria-selected="false"
					onmousedown={() => select(r)}
				>
					<span class="result-name">{r.name}</span>
					{#if r.admin1 || r.country}
						<span class="result-sub">{[r.admin1, r.country].filter(Boolean).join(', ')}</span>
					{/if}
				</button>
			{/each}
		</div>
	{/if}
</div>

<style>
	.city-search {
		position: relative;
	}

	.input-wrap {
		position: relative;
		display: flex;
		align-items: center;
	}

	.input-wrap input {
		padding-right: 28px;
	}

	.spinner {
		position: absolute;
		right: 10px;
		color: var(--text-muted);
		font-size: 14px;
		animation: spin 1s linear infinite;
		pointer-events: none;
	}

	@keyframes spin {
		from { transform: rotate(0deg); }
		to   { transform: rotate(360deg); }
	}

	.dropdown {
		position: absolute;
		top: calc(100% + 4px);
		left: 0;
		right: 0;
		background: var(--surface);
		border: 1px solid var(--border-subtle);
		border-radius: 8px;
		overflow: hidden;
		z-index: 100;
		box-shadow: 0 8px 24px rgba(0,0,0,0.4);
	}

	.result {
		width: 100%;
		display: flex;
		flex-direction: column;
		align-items: flex-start;
		gap: 1px;
		padding: 9px 12px;
		background: none;
		border: none;
		border-bottom: 1px solid rgba(255,255,255,0.06);
		cursor: pointer;
		text-align: left;
		transition: background 0.12s;
	}

	.result:last-child {
		border-bottom: none;
	}

	.result:hover {
		background: rgba(129, 140, 248, 0.15);
	}

	.result-name {
		font-size: 13px;
		color: var(--text);
		font-weight: 500;
	}

	.result-sub {
		font-size: 11px;
		color: var(--text-muted);
	}
</style>
