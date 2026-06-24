<script lang="ts">
  // A labelled container that structures a set of related options: a header
  // (headline + a master Toggle) above a slotted content area. The master
  // toggle is the group's "enabled" flag: when off the content is not rendered.
  import type { Snippet } from 'svelte';
  import Toggle from '../primitives/Toggle.svelte';

  interface Props {
    label: string;
    checked?: boolean;
    disabled?: boolean;
    children?: Snippet;
  }

  let {
    label,
    checked = $bindable(false),
    disabled = false,
    children,
  }: Props = $props();
</script>

<div class="og" class:checked>
  <div class="og-header">
    <span class="og-label">{label}</span>
    <Toggle bind:checked {disabled} ariaLabel="{label} (master enable)" />
  </div>
  {#if checked}
    <div class="og-content">
      {@render children?.()}
    </div>
  {/if}
</div>

<style>
  .og {
    display: block;
    padding: 12px 14px;
    --wails-draggable: none;
    user-select: none;
    -webkit-user-select: none;
  }
  .og-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 10px;
  }
  .og-label {
    font-size: 13px;
    font-weight: 600;
    color: var(--text);
  }
  /* Rendered only when checked (see markup), so nothing clips — no
     overflow:hidden / max-height cap, focus rings and tall fields show fully. */
  .og-content {
    padding-top: 10px;
  }
</style>
