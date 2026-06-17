<script lang="ts">
	import { onMount } from 'svelte';
	import { api } from '$lib/api';
	import type { Widget, DockerData } from '$lib/types';

	let { widget }: { widget: Widget } = $props();

	let data = $state<DockerData | null>(null);
	let error = $state('');
	let loading = $state(true);

	const endpoint   = $derived((widget.config?.endpoint as string) || 'unix:///var/run/docker.sock');
	const showStopped = $derived(!!widget.config?.show_stopped);
	const maxItems   = $derived(Number(widget.config?.max_items) || 10);

	const visibleContainers = $derived(
		(data?.containers ?? [])
			.filter(c => showStopped || c.state === 'running')
			.slice(0, maxItems)
	);

	async function fetchData() {
		loading = true;
		error = '';
		try {
			data = await api.getDocker(endpoint, showStopped, maxItems);
		} catch (e) {
			error = e instanceof Error ? e.message : 'Verbindungsfehler';
		} finally {
			loading = false;
		}
	}

	onMount(() => {
		fetchData();
		const interval = setInterval(fetchData, 30 * 1000);
		return () => clearInterval(interval);
	});

	$effect(() => {
		endpoint; showStopped; maxItems;
		fetchData();
	});

	function stateColor(state: string): string {
		switch (state) {
			case 'running':  return '#4ade80';
			case 'exited':   return '#f87171';
			case 'paused':   return '#fbbf24';
			case 'restarting': return '#60a5fa';
			default:         return '#94a3b8';
		}
	}

	function stateLabel(state: string): string {
		const labels: Record<string, string> = {
			running:    'läuft',
			exited:     'gestoppt',
			paused:     'pausiert',
			restarting: 'neustart',
			created:    'erstellt',
			dead:       'abgestürzt',
		};
		return labels[state] ?? state;
	}
</script>

{#if loading}
	<div class="center muted">Lade Docker…</div>
{:else if error}
	<div class="center error">{error}</div>
{:else if data}
	<div class="docker">
		<div class="stats">
			<div class="stat running">
				<span class="stat-dot" style="background:#4ade80"></span>
				<span class="stat-num">{data.running}</span>
				<span class="stat-label">aktiv</span>
			</div>
			<div class="stat-divider"></div>
			<div class="stat stopped">
				<span class="stat-dot" style="background:#f87171"></span>
				<span class="stat-num">{data.stopped}</span>
				<span class="stat-label">gestoppt</span>
			</div>
			<div class="stat-divider"></div>
			<div class="stat total">
				<span class="stat-num muted">{data.total}</span>
				<span class="stat-label">gesamt</span>
			</div>
		</div>

		{#if visibleContainers.length > 0}
			<div class="container-list">
				{#each visibleContainers as c (c.id)}
					<div class="container-row">
						<span class="dot" style="background:{stateColor(c.state)}"></span>
						<span class="cname">{c.name || c.id}</span>
						<span class="cstate" style="color:{stateColor(c.state)}">{stateLabel(c.state)}</span>
					</div>
					{#if c.ports?.length}
						<div class="cports">{c.ports.join(' · ')}</div>
					{/if}
				{/each}
			</div>
		{:else}
			<div class="center muted" style="flex:1">Keine Container</div>
		{/if}
	</div>
{/if}

<style>
	.center {
		display: flex;
		align-items: center;
		justify-content: center;
		height: 100%;
		font-size: 13px;
	}
	.muted  { color: var(--text-muted); }
	.error  { color: #f87171; }

	.docker {
		display: flex;
		flex-direction: column;
		gap: 10px;
		height: 100%;
	}

	.stats {
		display: flex;
		align-items: center;
		gap: 12px;
	}

	.stat {
		display: flex;
		align-items: center;
		gap: 6px;
	}

	.stat-dot {
		width: 8px;
		height: 8px;
		border-radius: 50%;
		flex-shrink: 0;
	}

	.stat-num {
		font-size: 22px;
		font-weight: 700;
		line-height: 1;
		color: var(--text);
	}

	.stat-num.muted { color: var(--text-muted); }

	.stat-label {
		font-size: 11px;
		color: var(--text-muted);
		align-self: flex-end;
		padding-bottom: 2px;
	}

	.stat-divider {
		width: 1px;
		height: 28px;
		background: rgba(255,255,255,0.1);
	}

	.container-list {
		display: flex;
		flex-direction: column;
		gap: 3px;
		overflow-y: auto;
		flex: 1;
	}

	.container-row {
		display: flex;
		align-items: center;
		gap: 7px;
		font-size: 12px;
	}

	.dot {
		width: 7px;
		height: 7px;
		border-radius: 50%;
		flex-shrink: 0;
	}

	.cname {
		flex: 1;
		color: var(--text);
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}

	.cstate {
		font-size: 10px;
		flex-shrink: 0;
	}

	.cports {
		font-size: 10px;
		color: var(--text-muted);
		padding-left: 14px;
		margin-top: -2px;
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}
</style>
