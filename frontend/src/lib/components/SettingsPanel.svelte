<script lang="ts">
	import { dashboard } from '$lib/stores/dashboard.svelte';
	import type { Theme } from '$lib/types';
	import WallpaperPicker from './WallpaperPicker.svelte';

	let theme = $state<Theme>(
		dashboard.config?.theme ?? {
			background_image: '',
			background_color: '#111111',
			card_color: 'rgba(28,28,28,0.97)',
			primary_color: '#818cf8',
			accent_color: '#fb923c',
			text_color: '#e8eaed',
			glass_effect: true,
			border_radius: 14,
			font: 'Inter',
		}
	);

	$effect(() => {
		if (dashboard.config?.theme) {
			theme = { ...dashboard.config.theme };
		}
	});

	function readPanelWidth() {
		try { return parseInt(localStorage.getItem('editor-panel-width') || '420', 10) || 420; }
		catch { return 420; }
	}

	let panelWidth = $state(readPanelWidth());
	let resizeDragging = $state(false);

	function startResize(e: PointerEvent) {
		resizeDragging = true;
		const startX = e.clientX;
		const startW = panelWidth;
		const onMove = (ev: PointerEvent) => {
			panelWidth = Math.max(300, Math.min(window.innerWidth * 0.9, startW + (startX - ev.clientX)));
		};
		const onUp = () => {
			resizeDragging = false;
			try { localStorage.setItem('editor-panel-width', String(panelWidth)); } catch { /* */ }
			window.removeEventListener('pointermove', onMove);
			window.removeEventListener('pointerup', onUp);
		};
		window.addEventListener('pointermove', onMove);
		window.addEventListener('pointerup', onUp);
		e.preventDefault();
	}

	async function save() {
		(document.activeElement as HTMLElement | null)?.blur();
		await dashboard.updateTheme(theme);
		dashboard.settingsOpen = false;
	}

	function discard() {
		(document.activeElement as HTMLElement | null)?.blur();
		dashboard.settingsOpen = false;
	}

	const fonts = ['Inter', 'Roboto', 'DM Sans', 'Nunito', 'Fira Code', 'JetBrains Mono'];

	const DEFAULT_THEME: Theme = {
		background_image: '',
		background_color: '#111111',
		card_color: 'rgba(28,28,28,0.97)',
		primary_color: '#818cf8',
		accent_color: '#fb923c',
		text_color: '#e8eaed',
		glass_effect: true,
		border_radius: 14,
		font: 'Inter',
	};

	function resetTheme() {
		theme = { ...DEFAULT_THEME };
	}

	let _pointerDownInPanel = false;
</script>

<div class="panel-overlay"
	onpointerdown={() => { _pointerDownInPanel = false; }}
	onclick={() => { if (!_pointerDownInPanel) save(); _pointerDownInPanel = false; }}
	onkeydown={(e) => { if (e.key === 'Escape') save(); }}
	role="dialog" aria-modal="true" tabindex="-1">
	<div class="panel" style="width:{panelWidth}px"
		onpointerdown={(e) => { _pointerDownInPanel = true; e.stopPropagation(); }}
		onclick={(e) => e.stopPropagation()}
		role="presentation">
		<div class="panel-resize-handle" class:dragging={resizeDragging} onpointerdown={startResize} role="separator" tabindex="-1" aria-label="Panel-Größe anpassen"></div>
		<div class="panel-header">
			<span>Design-Einstellungen</span>
			<button class="close-btn" onclick={save}>✕</button>
		</div>

		<div class="panel-body">
			<section>
				<h3>Hintergrund</h3>

				<div class="form-group">
					<label for="settings-bg-color">Hintergrundfarbe</label>
					<div class="color-row">
						<input id="settings-bg-color" type="color" class="color-picker" bind:value={theme.background_color} />
						<input type="text" bind:value={theme.background_color} placeholder="#0f172a" aria-label="Hintergrundfarbe als Hex" />
					</div>
				</div>

				<div class="form-group">
					<span class="label-text">Hintergrundbild</span>
					<WallpaperPicker
						value={theme.background_image}
						onchange={(url) => (theme.background_image = url)}
					/>
				</div>

				<div class="form-group">
					<label for="settings-bg-url">Oder externe URL</label>
					<input id="settings-bg-url" type="url" bind:value={theme.background_image} placeholder="https://..." />
				</div>
			</section>

			<section>
				<h3>Farben</h3>

				<div class="color-grid">
					<div class="form-group">
						<label for="settings-card-color">Karten-Farbe</label>
						<div class="color-row">
							<input id="settings-card-color" type="color" class="color-picker"
								value={theme.card_color.startsWith('#') ? theme.card_color : '#1c1c1c'}
								oninput={(e) => theme.card_color = (e.target as HTMLInputElement).value}
							/>
							<input type="text" bind:value={theme.card_color} aria-label="Karten-Farbe als Hex" />
						</div>
					</div>

					<div class="form-group">
						<label for="settings-primary-color">Primärfarbe</label>
						<div class="color-row">
							<input id="settings-primary-color" type="color" class="color-picker" bind:value={theme.primary_color} />
							<input type="text" bind:value={theme.primary_color} aria-label="Primärfarbe als Hex" />
						</div>
					</div>

					<div class="form-group">
						<label for="settings-accent-color">Akzentfarbe</label>
						<div class="color-row">
							<input id="settings-accent-color" type="color" class="color-picker" bind:value={theme.accent_color} />
							<input type="text" bind:value={theme.accent_color} aria-label="Akzentfarbe als Hex" />
						</div>
					</div>

					<div class="form-group">
						<label for="settings-text-color">Textfarbe</label>
						<div class="color-row">
							<input id="settings-text-color" type="color" class="color-picker" bind:value={theme.text_color} />
							<input type="text" bind:value={theme.text_color} aria-label="Textfarbe als Hex" />
						</div>
					</div>
				</div>
			</section>

			<section>
				<h3>Stil</h3>

				<div class="form-group">
					<span class="label-text">Glaseffekt (Blur)</span>
					<label class="toggle">
						<input type="checkbox" bind:checked={theme.glass_effect} />
						<span class="toggle-track"></span>
					</label>
				</div>

				<div class="form-group">
					<label for="settings-border-radius">Eckenradius: {theme.border_radius}px</label>
					<input id="settings-border-radius" type="range" min="0" max="28" bind:value={theme.border_radius} />
				</div>

				<div class="form-group">
					<label for="settings-font">Schriftart</label>
					<select id="settings-font" bind:value={theme.font}>
						{#each fonts as f}
							<option value={f}>{f}</option>
						{/each}
					</select>
				</div>
			</section>
		</div>

		<div class="panel-footer">
			{#if dashboard.authenticated}
				<button class="btn btn-ghost logout-btn" onclick={() => dashboard.logout()}>Abmelden</button>
			{/if}
			<div class="footer-right">
				<button class="btn btn-ghost reset-theme-btn" onclick={resetTheme} title="Alle Designwerte auf Standard zurücksetzen">
					Zurücksetzen
				</button>
				<button class="btn btn-ghost" onclick={discard}>Verwerfen</button>
				<button class="btn btn-primary" onclick={save}>Speichern</button>
			</div>
		</div>
	</div>
</div>

<style>
	h3 {
		font-size: 11px;
		font-weight: 600;
		letter-spacing: 0.08em;
		text-transform: uppercase;
		color: var(--text-muted);
		margin-bottom: 14px;
	}

	section {
		margin-bottom: 28px;
	}

	.color-row {
		display: flex;
		align-items: center;
		gap: 8px;
	}

	.color-picker {
		width: 40px;
		height: 36px;
		padding: 2px;
		flex-shrink: 0;
		cursor: pointer;
		border-radius: 6px;
	}

	.color-grid {
		display: grid;
		grid-template-columns: 1fr 1fr;
		gap: 0 16px;
	}

	.toggle {
		display: flex;
		align-items: center;
		cursor: pointer;
		margin-bottom: 0;
		color: var(--text);
	}

	.toggle input { display: none; }

	.toggle-track {
		width: 40px;
		height: 22px;
		background: rgba(255,255,255,0.15);
		border-radius: 11px;
		position: relative;
		transition: background 0.2s;
	}

	.toggle-track::after {
		content: '';
		position: absolute;
		left: 3px;
		top: 3px;
		width: 16px;
		height: 16px;
		border-radius: 50%;
		background: #fff;
		transition: transform 0.2s;
	}

	.toggle input:checked ~ .toggle-track {
		background: var(--primary);
	}

	.toggle input:checked ~ .toggle-track::after {
		transform: translateX(18px);
	}

	input[type="range"] {
		padding: 0;
		height: 4px;
		accent-color: var(--primary);
	}

	.panel-footer {
		padding: 16px 24px;
		border-top: 1px solid rgba(255,255,255,0.07);
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 8px;
	}

	.footer-right {
		display: flex;
		gap: 8px;
	}

	.logout-btn {
		font-size: 12px;
		color: #f87171;
		border-color: rgba(248, 113, 113, 0.3);
	}

	.logout-btn:hover {
		background: rgba(248, 113, 113, 0.1);
	}

	.reset-theme-btn {
		font-size: 12px;
		color: var(--text-muted);
	}

	.reset-theme-btn:hover {
		color: #f87171;
		border-color: rgba(248, 113, 113, 0.3);
		background: rgba(248, 113, 113, 0.08);
	}

	.close-btn {
		background: none;
		border: none;
		color: var(--text-muted);
		font-size: 18px;
		cursor: pointer;
		line-height: 1;
	}

	.close-btn:hover {
		color: var(--text);
	}
</style>
