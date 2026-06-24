<script lang="ts">
  import type { DisplayAlbum } from '../../lib/preview';

  interface Props {
    album: DisplayAlbum;
    onedit?: () => void;
  }

  let { album, onedit }: Props = $props();

  // Genre and year share one line: "genre · year" (whichever exist).
  const subLine = $derived(
    [album.genre, album.year].filter(Boolean).join(' · '),
  );
</script>

<div class="ah">
  <div class="ah-art">
    {#if album.artwork}<img src={album.artwork} alt="" />{/if}
    {#if album.staged}<span class="ah-badge" title="Has unsaved edits"></span>{/if}
  </div>
  <div class="ah-meta">
    <div class="ah-title">{album.title || 'Unknown Album'}</div>
    {#if album.artist}<div class="ah-artist">{album.artist}</div>{/if}
    {#if subLine}<div class="ah-genre">{subLine}</div>{/if}
  </div>
  <button
    type="button"
    class="ah-edit"
    title="Edit metadata"
    onclick={(e) => {
      e.preventDefault();
      e.stopPropagation();
      onedit?.();
    }}
  >
    Edit
  </button>
</div>

<style>
  .ah {
    position: relative;
    display: flex;
    align-items: center;
    gap: 12px;
    margin-bottom: 8px;
  }
  .ah-art {
    position: relative;
    width: 44px;
    height: 44px;
    flex: none;
    border-radius: 5px;
    overflow: hidden;
    background: var(--panel-inset);
    border: 0.5px solid var(--hairline);
  }
  .ah-art img {
    width: 100%;
    height: 100%;
    object-fit: cover;
    display: block;
  }
  .ah-badge {
    position: absolute;
    top: 3px;
    right: 3px;
    width: 8px;
    height: 8px;
    border-radius: 50%;
    background: var(--accent);
    border: 1.5px solid var(--panel);
    box-shadow: 0 0 0 1px var(--hairline);
  }
  .ah-title {
    color: var(--text);
    font-size: 13px;
    font-weight: 600;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .ah-artist {
    color: var(--text-dim);
    font-size: 12px;
    margin-top: 1px;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .ah-genre {
    color: var(--text-faint);
    font-size: 11px;
    margin-top: 1px;
  }
  .ah-edit {
    position: absolute;
    top: -3px;
    right: -3px;
    display: flex;
    align-items: center;
    justify-content: center;
    height: 20px;
    padding: 0 8px;
    border: 0.5px solid var(--hairline);
    border-radius: 5px;
    background: var(--panel);
    color: var(--text-dim);
    font-size: 11px;
    font-weight: 500;
    cursor: pointer;
    opacity: 0;
    transition: opacity 0.12s ease;
  }
  .ah:hover .ah-edit,
  .ah-edit:focus-visible {
    opacity: 1;
  }
  .ah-edit:hover {
    color: var(--text);
    border-color: var(--accent);
  }
</style>
