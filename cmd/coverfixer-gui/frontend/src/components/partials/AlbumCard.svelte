<script lang="ts">
  // Composes an AlbumHeader + a disc-grouped track list from a display album.
  // Equivalent to the old <preview-album> custom element.
  import type { DisplayAlbum } from '../../lib/preview';
  import { discGroups } from '../../lib/preview';
  import AlbumHeader from './AlbumHeader.svelte';
  import TrackRow from './TrackRow.svelte';

  interface Props {
    album: DisplayAlbum;
    onedit?: () => void;
  }

  let { album, onedit }: Props = $props();

  // Tracks arrive sorted by the backend (disc, number, title). Disc headings
  // render only when the album spans more than one disc.
  const groups = $derived(discGroups(album.tracks));
</script>

<section class="pa">
  <AlbumHeader {album} {onedit} />
  <div class="pa-body">
    {#each groups as g (g.disc)}
      {#if groups.length > 1}
        <div class="pa-disc">Disc {g.disc}</div>
      {/if}
      <ol class="pa-ol">
        {#each g.tracks as t, i (i)}
          <TrackRow track={t} />
        {/each}
      </ol>
    {/each}
  </div>
</section>

<style>
  .pa {
    padding: 10px 12px;
    background: var(--panel);
    border: 0.5px solid var(--hairline);
    border-radius: var(--radius);
  }
  .pa-ol {
    margin: 0;
    padding-left: 22px;
    list-style: none;
  }
  .pa-disc {
    color: var(--text-dim);
    font-size: 11px;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.04em;
    margin: 8px 0 2px;
    padding: 4px 0 4px calc(44px + 2ch);
  }
  .pa-body > .pa-disc:first-child {
    margin-top: 0;
  }
</style>
