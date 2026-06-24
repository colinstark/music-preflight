<script lang="ts">
  import { store } from '../../lib/store.svelte';
  import OptionGroup from '../partials/OptionGroup.svelte';
  import TabBar from '../primitives/TabBar.svelte';
  import TextField from '../primitives/TextField.svelte';
  import ActionBar from './ActionBar.svelte';
  import Output from './Output.svelte';

  let disabled = $derived(store.running);
</script>

<aside class="sidebar">
  <div class="groups">
    <OptionGroup label="Resize Cover Art" bind:checked={store.form.resizeEmbedded} {disabled}>
      <label class="field">
        <span>Size</span>
        <TabBar bind:value={store.form.artSize} options="500,480,320,240" {disabled} />
      </label>
    </OptionGroup>

    <OptionGroup label="Extract cover.jpg" bind:checked={store.form.coverJpgGroup} {disabled}>
      <label class="field">
        <span>Size</span>
        <TabBar bind:value={store.form.coverJpgSize} options="500,480,320,240" {disabled} />
      </label>
    </OptionGroup>

    <OptionGroup label="Transcode" bind:checked={store.form.transcodeGroup} {disabled}>
      <div class="fields">
        <label class="field">
          <span>Format</span>
          <TabBar bind:value={store.form.transcodeFormat} options="MP3:mp3,AAC:aac" {disabled} />
        </label>
        <label class="field">
          <span>Quality</span>
          <TabBar bind:value={store.form.transcodeQuality} options="320,256,192" {disabled} />
        </label>
      </div>
    </OptionGroup>

    <OptionGroup label="Update Metadata for all" bind:checked={store.form.metadataGroup} {disabled}>
      <div class="meta-fields">
        <label class="field">
          <span>Album Artist</span>
          <TextField
            id="albumArtist"
            bind:value={store.form.albumArtist}
            placeholder="prefilled on folder pick"
            {disabled}
          />
        </label>
        <label class="field">
          <span>Genre</span>
          <TextField
            id="genre"
            bind:value={store.form.genre}
            placeholder="prefilled on folder pick"
            {disabled}
          />
        </label>
      </div>
    </OptionGroup>
  </div>

  <ActionBar />
  <Output />
</aside>

<style>
  .sidebar {
    display: flex;
    flex-direction: column;
    gap: 14px;
    padding: 16px 18px;
  }
  .groups {
    background: var(--panel);
    border: 0.5px solid var(--hairline);
    border-radius: var(--radius);
    overflow: hidden;
  }
  /* Hairline dividers between flush option-groups. */
  .groups > :global(.og) + :global(.og) {
    border-top: 0.5px solid var(--hairline);
  }
  .fields {
    display: flex;
    gap: 14px;
    flex-wrap: wrap;
    margin-top: 4px;
  }
  .field {
    display: flex;
    flex-direction: column;
    gap: 4px;
    font-size: 11px;
    color: var(--text-dim);
  }
  .meta-fields {
    display: flex;
    gap: 12px;
  }
  .meta-fields .field {
    flex: 1 1 0;
  }
</style>
