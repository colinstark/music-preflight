<script lang="ts">
  import { onMount } from 'svelte';
  import { store } from './lib/store.svelte';
  import TitleBar from './components/blocks/TitleBar.svelte';
  import EmptyState from './components/blocks/EmptyState.svelte';
  import FolderBar from './components/blocks/FolderBar.svelte';
  import Sidebar from './components/blocks/Sidebar.svelte';
  import Library from './components/blocks/Library.svelte';
  import EditAlbumModal from './components/blocks/EditAlbumModal.svelte';

  // Connect to Wails globals + events and seed the form once mounted. The store
  // drives all view state from here on.
  onMount(() => {
    void store.init();
  });
</script>

<TitleBar />
<main class="app">
  {#if store.phase === 'empty'}
    <EmptyState />
  {:else}
    <FolderBar />
    <div class="content">
      <Sidebar />
      <Library />
    </div>
  {/if}
</main>

<EditAlbumModal />

<style>
  .app {
    padding: 0;
    flex: 1 1 auto;
    min-height: 0;
    display: flex;
    flex-direction: column;
    --wails-draggable: drag;
  }
  .content {
    flex: 1 1 auto;
    min-height: 0;
    overflow-y: auto;
    display: flex;
    flex-direction: column;
    gap: 14px;
  }

  /* Wide layout: sidebar on the left, library on the right, inside the
     scrollable content. Both scroll together; the sidebar sticks. The selectors
     are global because the grid spans child components. */
  @media (min-width: 640px) {
    .content {
      display: grid;
      grid-template-columns: minmax(260px, 320px) minmax(0, 1fr);
      grid-template-areas: 'sidebar library';
      align-items: start;
      column-gap: 14px;
    }
    .content :global(.sidebar) {
      grid-area: sidebar;
      align-self: start;
      position: sticky;
      top: 0;
      padding: 16px 0 16px 18px;
    }
    .content :global(#libraryBlock) {
      grid-area: library;
      padding: 16px 18px 16px 0;
    }
  }
</style>
