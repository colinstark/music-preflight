<script lang="ts">
  // Custom header band: solid (covers the body vibrancy), draggable,
  // theme-aware. The native traffic lights overlay its top-left on macOS.
</script>

<header class="titlebar"><span class="titlebar-title">Music Preflight</span></header>

<style>
  .titlebar {
    flex: none;
    height: 28px;
    display: flex;
    align-items: center;
    justify-content: center;
    background: var(--titlebar-bg);
    color: var(--titlebar-fg);
    font-size: 13px;
    font-weight: 600;
    --wails-draggable: drag;
    user-select: none;
    -webkit-user-select: none;
  }
</style>
