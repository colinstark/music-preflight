<script lang="ts">
  import Button from '../primitives/Button.svelte';
  import { store } from '../../lib/store.svelte';
</script>

<section class="folder-row">
  <Button onclick={() => store.chooseFolder()} disabled={store.running}>
    Choose Folder
  </Button>
  <svg class="folder-icon" viewBox="0 0 24 24" fill="currentColor">
    <path
      d="M12.4142 5H21C21.5523 5 22 5.44772 22 6V20C22 20.5523 21.5523 21 21 21H3C2.44772 21 2 20.5523 2 20V4C2 3.44772 2.44772 3 3 3H10.4142L12.4142 5Z"
    ></path>
  </svg>
  <span class="path" title={store.dir}>{store.folderDisplayName || 'No folder selected'}</span>
  <span class="folder-counts">{store.countsText}</span>
</section>

<style>
  .folder-row {
    flex: none;
    display: flex;
    align-items: center;
    gap: 10px;
    padding: 10px 18px;
    background: var(--folder-bar-bg);
    border-bottom: 0.5px solid var(--hairline);
    --wails-draggable: drag;
  }
  .folder-icon {
    width: 15px;
    height: 15px;
    flex: none;
    color: var(--text-dim);
  }
  .path {
    color: var(--text);
    font-size: 13px;
    font-weight: 500;
    flex: 0 1 auto;
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    --wails-draggable: drag;
  }
  .folder-counts {
    flex: none;
    white-space: nowrap;
    color: var(--text-faint);
    font-size: 12px;
    font-variant-numeric: tabular-nums;
  }
  .folder-counts:empty {
    display: none;
  }
</style>
