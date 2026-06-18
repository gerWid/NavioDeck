<script lang="ts">
	import { onMount, onDestroy, tick } from 'svelte';
	import { GridStack } from 'gridstack';
	import 'gridstack/dist/gridstack.min.css';
	import type { Widget } from '$lib/types';
	import { dashboard } from '$lib/stores/dashboard.svelte';
	import WidgetWrapper from './WidgetWrapper.svelte';
	import { mount, unmount } from 'svelte';

	let { widgets }: { widgets: Widget[] } = $props();

	let gridEl: HTMLElement;
	let grid: GridStack;
	const mounted = new Map<string, ReturnType<typeof mount>>();
	let prevIds = new Set<string>();

	// Guard so the add/remove $effect only fires after onMount has set up prevIds
	let initialized = $state(false);

	// Sets CSS `order` on a grid item so the mobile flex-column layout
	// respects the visual desktop order (top→bottom, left→right).
	function setOrder(id: string, x: number, y: number) {
		const el = gridEl.querySelector(`[gs-id="${id}"]`) as HTMLElement | null;
		if (el) el.style.order = String(y * 100 + x);
	}

	function addWidgetToGrid(widget: Widget) {
		if (mounted.has(widget.id)) return;

		grid.addWidget({
			id: widget.id,
			x: widget.position.x,
			y: widget.position.y,
			w: widget.position.w,
			h: widget.position.h,
			content: `<div id="wc-${widget.id}" style="height:100%"></div>`,
		});

		// Wait for GridStack to write the content to the DOM
		tick().then(() => {
			const container = document.getElementById(`wc-${widget.id}`);
			if (container && !mounted.has(widget.id)) {
				const component = mount(WidgetWrapper, { target: container, props: { widgetId: widget.id } });
				mounted.set(widget.id, component);
			}
			setOrder(widget.id, widget.position.x, widget.position.y);
		});
	}

	function removeWidgetFromGrid(id: string) {
		const component = mounted.get(id);
		if (component) {
			unmount(component);
			mounted.delete(id);
		}
		const el = gridEl.querySelector(`[gs-id="${id}"]`);
		if (el) grid.removeWidget(el as HTMLElement, true);
	}

	onMount(async () => {
		await tick();

		// gridstack v11+ inserts widget `content` via textContent (XSS-safe) by default,
		// which would render our container markup as escaped text. We mount Svelte
		// components into that container ourselves, so restore HTML rendering here.
		GridStack.renderCB = (el: HTMLElement, w: { content?: string }) => {
			el.innerHTML = w.content ?? '';
		};

		grid = GridStack.init(
			{
				float: false,
				cellHeight: 80,
				column: 12,
				margin: 8,
				// No handle restriction: the edit overlay (pointer-events:all) makes
				// the whole card a drag surface; buttons stop propagation themselves
				disableDrag: true,
				disableResize: true,
				resizable: { handles: 'se' },
			},
			gridEl
		);

		grid.on('change', (_: Event, items: any[]) => {
			if (!dashboard.editMode) return;
			const layout = items.map((item) => ({
				id: item.id as string,
				x: item.x ?? 0,
				y: item.y ?? 0,
				w: item.w ?? 2,
				h: item.h ?? 2,
			}));
			dashboard.saveLayout(layout);
			// Keep mobile CSS order in sync after drag
			for (const item of layout) {
				setOrder(item.id, item.x, item.y);
			}
		});

		// Add all initial widgets imperatively — no {#each} conflict with GridStack
		for (const widget of widgets) {
			addWidgetToGrid(widget);
		}

		prevIds = new Set(widgets.map((w) => w.id));
		initialized = true;
	});

	onDestroy(() => {
		for (const [, component] of mounted) {
			unmount(component);
		}
		mounted.clear();
		grid?.destroy(false);
	});

	// Toggle drag/resize with edit mode.
	// Drag/resize is disabled on mobile since touch drag in GridStack is unreliable
	// and the grid reflows to a single flex column via CSS anyway.
	$effect(() => {
		if (!initialized) return;
		const mobile = window.innerWidth < 768;
		grid.enableMove(dashboard.editMode && !mobile);
		grid.enableResize(dashboard.editMode && !mobile);
	});

	// Handle widget additions and removals after initial mount
	$effect(() => {
		if (!initialized) return;

		const currentIds = new Set(widgets.map((w) => w.id));

		for (const id of prevIds) {
			if (!currentIds.has(id)) removeWidgetFromGrid(id);
		}
		for (const widget of widgets) {
			if (!prevIds.has(widget.id)) addWidgetToGrid(widget);
		}

		prevIds = currentIds;
	});
</script>

<!-- Grid is fully managed by GridStack via the imperative API above -->
<div bind:this={gridEl} class="grid-stack" class:edit-mode={dashboard.editMode}></div>
