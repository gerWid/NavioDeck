<script lang="ts">
	import { dashboard } from '$lib/stores/dashboard.svelte';

	let password = $state('');
	let error    = $state('');
	let loading  = $state(false);

	async function submit(e: Event) {
		e.preventDefault();
		if (!password || loading) return;
		loading = true;
		error   = '';
		const err = await dashboard.login(password);
		loading = false;
		if (err) {
			error    = err;
			password = '';
		}
	}

	function focusNode(node: HTMLElement) {
		node.focus();
	}
</script>

<div class="overlay">
	<div class="card">
		<div class="logo">🏠</div>
		<h1>NavioDeck</h1>
		<p class="subtitle">Bitte melde dich an</p>

		<form onsubmit={submit}>
			<div class="field">
				<input
					type="password"
					bind:value={password}
					placeholder="Passwort"
					autocomplete="current-password"
					use:focusNode
					disabled={loading}
				/>
			</div>
			{#if error}
				<p class="error">{error}</p>
			{/if}
			<button type="submit" class="btn-login" disabled={loading || !password}>
				{loading ? 'Prüfe…' : 'Anmelden'}
			</button>
		</form>
	</div>
</div>

<style>
	.overlay {
		position: fixed;
		inset: 0;
		background: var(--bg-color, #111111);
		display: flex;
		align-items: center;
		justify-content: center;
		z-index: 100;
	}

	.card {
		background: var(--surface);
		border: 1px solid var(--border-subtle);
		border-radius: 20px;
		padding: 40px 36px 36px;
		width: 340px;
		max-width: 95vw;
		text-align: center;
		backdrop-filter: blur(12px);
	}

	.logo {
		font-size: 40px;
		margin-bottom: 12px;
	}

	h1 {
		font-size: 20px;
		font-weight: 700;
		color: var(--text, #f1f5f9);
		margin: 0 0 6px;
	}

	.subtitle {
		font-size: 13px;
		color: var(--text-muted, #94a3b8);
		margin: 0 0 28px;
	}

	.field input {
		width: 100%;
		text-align: center;
		font-size: 15px;
		letter-spacing: 0.1em;
	}

	.error {
		font-size: 12px;
		color: #f87171;
		margin: 6px 0 0;
		text-align: center;
	}

	.btn-login {
		margin-top: 16px;
		width: 100%;
		padding: 10px;
		background: var(--primary, #818cf8);
		color: #fff;
		border: none;
		border-radius: 10px;
		font-size: 14px;
		font-weight: 600;
		cursor: pointer;
		transition: opacity 0.15s;
		font-family: var(--font);
	}

	.btn-login:disabled {
		opacity: 0.5;
		cursor: default;
	}

	.btn-login:not(:disabled):hover {
		opacity: 0.88;
	}
</style>
