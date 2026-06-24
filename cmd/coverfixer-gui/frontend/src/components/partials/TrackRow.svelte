<script lang="ts">
  import { formatDuration } from '../../lib/preview';
  import type { DisplayTrack } from '../../lib/preview';

  interface Props {
    track: DisplayTrack;
  }

  let { track }: Props = $props();
</script>

<li class="at">
  <span class="at-n">{track.number ? track.number + '.' : ''}</span>
  <span class="at-title">
    {track.title}
    {#if track.artist}<span class="at-artist"> — {track.artist}</span>{/if}
  </span>
  <span class="at-dur">{formatDuration(track.duration)}</span>
</li>

<style>
  .at {
    display: grid;
    grid-template-columns: 2ch 1fr auto;
    column-gap: 8px;
    align-items: baseline;
    padding: 1px 0 1px 12px;
    margin-bottom: 4px;
  }
  .at-n {
    color: var(--text-faint);
    font-variant-numeric: tabular-nums;
    text-align: right;
    font-size: 12px;
  }
  .at-title {
    color: var(--text-dim);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    font-size: 12.5px;
  }
  .at-artist {
    color: var(--text-faint);
    font-size: 12.5px;
  }
  .at-dur {
    color: var(--text-faint);
    font-variant-numeric: tabular-nums;
    font-size: 12px;
  }
</style>
