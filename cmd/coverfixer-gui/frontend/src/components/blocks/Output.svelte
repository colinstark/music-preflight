<script lang="ts">
  import { store } from '../../lib/store.svelte';
</script>

<section class="summary">
  {#if store.summary}<pre class="summary-text">{store.summary}</pre>{/if}
  {#if store.error}<pre class="error-text" role="alert">{store.error}</pre>{/if}
</section>

<style>
  .summary {
    display: flex;
    flex-direction: column;
    gap: 8px;
    --wails-draggable: none;
  }
  .summary-text {
    margin: 0;
    font-family: var(--mono);
    font-size: 12.5px;
    white-space: pre-wrap;
    color: var(--ok);
  }
  .error-text {
    margin: 0;
    font-family: var(--mono);
    font-size: 12.5px;
    white-space: pre-wrap;
    color: var(--danger);
    background: var(--danger-bg);
    border-radius: var(--radius-field);
    padding: 8px 10px;
  }
</style>
