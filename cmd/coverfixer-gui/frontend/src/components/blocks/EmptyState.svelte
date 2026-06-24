<script lang="ts">
  import Button from '../primitives/Button.svelte';
  import { store } from '../../lib/store.svelte';
</script>

<div class="empty-state">
  <Button variant="primary" size="large" onclick={() => store.chooseFolder()}>
    Choose Folder
  </Button>
</div>

<style>
  .empty-state {
    flex: 1 1 auto;
    min-height: 0;
    display: flex;
    align-items: center;
    justify-content: center;
  }
</style>
