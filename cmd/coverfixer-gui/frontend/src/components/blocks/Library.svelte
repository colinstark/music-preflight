<script lang="ts">
  import { store } from '../../lib/store.svelte';
  import AlbumCard from '../partials/AlbumCard.svelte';

  // Idle library preview. While the first load is in flight (albums === null)
  // show a placeholder; a later rescan keeps the old albums visible until the
  // fresh ones arrive (no empty-state flash).
  let albums = $derived(store.albums);
  let visible = $derived(store.visibleAlbums);
</script>

<section id="libraryBlock" class="progress" aria-label="Library">
  {#if albums === null}
    <div class="preview-empty">Reading library…</div>
  {:else if albums.length === 0}
    <div class="preview-empty">No audio files found.</div>
  {:else}
    <div class="preview">
      {#each visible as album, i (i)}
        <AlbumCard {album} onedit={() => store.openEdit(i)} />
      {/each}
    </div>
  {/if}
</section>

<style>
  #libraryBlock {
    display: block;
    --wails-draggable: none;
  }
  .preview {
    display: flex;
    flex-direction: column;
    gap: 10px;
  }
  .preview-empty {
    color: var(--text-faint);
    font-size: 12.5px;
    padding: 12px 4px;
  }
</style>
