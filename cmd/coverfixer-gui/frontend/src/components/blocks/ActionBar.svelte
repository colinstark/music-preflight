<script lang="ts">
  import { store } from '../../lib/store.svelte';
  import Button from '../primitives/Button.svelte';
  import Toggle from '../primitives/Toggle.svelte';

  // ActionBar: progress bar (indeterminate, shown while running), the idle-only
  // backup toggle, and Run/Cancel. Mirrors the original .actions row.
</script>

<section class="actions">
  {#if store.running}
    <div class="progress-bar"><div class="progress-bar-fill"></div></div>
  {/if}
  {#if !store.running}
    <Toggle bind:checked={store.form.backup} label="Create backup" />
  {/if}
  <Button variant="primary" disabled={!store.canRun} onclick={() => store.run()}>Run</Button>
  {#if store.running}
    <Button variant="cancel" onclick={() => store.cancel()}>Cancel</Button>
  {/if}
</section>

<style>
  .actions {
    display: flex;
    align-items: center;
    gap: 10px;
    justify-content: flex-end;
    --wails-draggable: none;
  }
  .progress-bar {
    flex: 1 1 auto;
    min-width: 80px;
    height: 6px;
    margin-right: 12px;
    align-self: center;
    background: var(--panel-inset);
    border: 0.5px solid var(--hairline-strong);
    border-radius: 999px;
    overflow: hidden;
  }
  .progress-bar-fill {
    height: 100%;
    width: 40%;
    background: var(--accent);
    border-radius: 999px;
    animation: progress-indeterminate 1.1s ease-in-out infinite;
  }
  @keyframes progress-indeterminate {
    0% {
      transform: translateX(-110%);
    }
    100% {
      transform: translateX(260%);
    }
  }
</style>
