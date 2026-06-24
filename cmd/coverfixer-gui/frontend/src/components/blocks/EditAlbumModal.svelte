<script lang="ts">
  import { store } from '../../lib/store.svelte';
  import Button from '../primitives/Button.svelte';
  import TextField from '../primitives/TextField.svelte';

  // Native <dialog> sheet driven by the store's editing state. Opening/
  // closing is a side effect of store.openEdit / store.closeEdit (called from
  // here and by AlbumCard's Edit button); this $effect keeps the DOM dialog in
  // step. Save stages (or unstages) the edit; Revert/Cancel/Esc just close.
  let dlg = $state<HTMLDialogElement>();

  $effect(() => {
    if (!dlg) return;
    const open = store.editingIdx !== null;
    if (open && !dlg.open) dlg.showModal();
    else if (!open && dlg.open) dlg.close();
  });

  let heading = $derived(
    store.editingIdx !== null && store.albums
      ? 'Edit — ' + (store.albums[store.editingIdx]?.title || 'Unknown Album')
      : 'Edit Album',
  );
  let statusText = $derived(
    store.editingIdx !== null && store.stagedEdits.has(store.editingIdx)
      ? 'Unsaved edits staged'
      : '',
  );
</script>

<dialog bind:this={dlg} class="edit-modal" onclose={() => store.closeEdit()}>
  {#if store.editingDraft}
    <form
      class="edit-form"
      onsubmit={(e) => {
        e.preventDefault();
        store.stageEdit();
      }}
    >
      <div class="edit-head">
        <h2 class="edit-title">{heading}</h2>
      </div>
      <p class="edit-hint">Staged until you Run. Apply-on-Run rules apply.</p>

      <fieldset class="edit-album">
        <legend>Album</legend>
        <label class="field"><span>Title</span><TextField bind:value={store.editingDraft.album} /></label>
        <label class="field">
          <span>Album Artist</span><TextField bind:value={store.editingDraft.albumArtist} />
        </label>
        <label class="field"><span>Genre</span><TextField bind:value={store.editingDraft.genre} /></label>
        <label class="field">
          <span>Year</span><TextField bind:value={store.editingDraft.year} inputmode="numeric" />
        </label>
      </fieldset>

      <fieldset class="edit-tracks">
        <legend>Tracks</legend>
        <div class="edit-track-head"><span>#</span><span>Title</span><span>Artist</span></div>
        <div class="edit-track-list">
          {#each store.editingDraft.tracks as t, i (i)}
            <label class="edit-track">
              <input class="et-num" type="number" min="0" bind:value={t.trackNumber} />
              <input class="et-title" type="text" bind:value={t.title} />
              <input class="et-artist" type="text" bind:value={t.artist} placeholder="(album artist)" />
            </label>
          {/each}
        </div>
      </fieldset>

      <div class="edit-actions">
        <Button onclick={() => store.revertEdit()}>Revert</Button>
        <span class="edit-status" class:ok={!!statusText}>{statusText}</span>
        <Button onclick={() => store.closeEdit()}>Cancel</Button>
        <Button variant="primary" type="submit">Save</Button>
      </div>
    </form>
  {/if}
</dialog>

<style>
  .edit-modal {
    margin: auto;
    padding: 0;
    border: 0.5px solid var(--hairline-strong);
    border-radius: var(--radius);
    background: var(--modal-bg);
    backdrop-filter: saturate(1.2);
    color: var(--text);
    width: min(560px, calc(100vw - 40px));
    max-height: calc(100vh - 60px);
    box-shadow: 0 20px 60px rgba(0, 0, 0, 0.45);
  }
  .edit-modal::backdrop {
    background: rgba(0, 0, 0, 0.45);
  }
  .edit-form {
    display: flex;
    flex-direction: column;
    min-height: 0;
    max-height: calc(100vh - 60px);
  }
  .edit-head {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 8px;
    padding: 14px 16px 4px;
  }
  .edit-title {
    margin: 0;
    font-size: 14px;
    font-weight: 600;
  }
  .edit-hint {
    margin: 0 16px 10px;
    color: var(--text-faint);
    font-size: 11px;
  }
  .edit-form :global(fieldset) {
    border: 0.5px solid var(--hairline);
    border-radius: var(--radius-field);
    margin: 0 16px 12px;
    padding: 8px 12px 10px;
  }
  .edit-form :global(legend) {
    font-size: 11px;
    color: var(--text-dim);
    padding: 0 4px;
  }
  .edit-album {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 8px 12px;
    flex: 0 0 auto;
  }
  .edit-album :global(.field) {
    width: auto;
  }
  .edit-tracks {
    display: flex;
    flex-direction: column;
    flex: 1 1 auto;
    min-height: 0;
  }
  .edit-track-head,
  .edit-track {
    display: grid;
    grid-template-columns: 40px minmax(0, 1fr) minmax(0, 1fr);
    column-gap: 8px;
    align-items: center;
  }
  .edit-track-head {
    margin: 4px 0 2px;
    font-size: 10.5px;
    color: var(--text-faint);
    text-transform: uppercase;
    letter-spacing: 0.04em;
    flex: 0 0 auto;
  }
  .edit-track-list {
    display: flex;
    flex-direction: column;
    gap: 5px;
    flex: 1 1 auto;
    min-height: 0;
    overflow-y: auto;
    padding-right: 2px;
  }
  .edit-track input {
    background: var(--field-bg);
    color: var(--text);
    border: 0.5px solid var(--hairline-strong);
    border-radius: var(--radius-field);
    padding: 3px 6px;
    font-size: 12.5px;
    width: 100%;
    min-width: 0;
    --wails-draggable: none;
  }
  .edit-track input:focus {
    outline: none;
    border-color: var(--accent);
    box-shadow: 0 0 0 3px var(--focus-ring);
  }
  .edit-track input.et-num {
    text-align: center;
    font-variant-numeric: tabular-nums;
  }
  .edit-actions {
    display: flex;
    align-items: center;
    gap: 10px;
    justify-content: flex-end;
    padding: 4px 16px 14px;
    flex: 0 0 auto;
  }
  .edit-status {
    flex: 1 1 auto;
    color: var(--text-faint);
    font-size: 11px;
  }
  .edit-status.ok {
    color: var(--ok);
  }
</style>
