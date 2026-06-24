<script lang="ts">
  // macOS-style sliding toggle. bind:checked two-ways with the caller's state.
  // Theme tokens (--accent, --switch-off, --focus-ring, --text-dim) are defined
  // on :root and inherit through the scoped boundary.
  interface Props {
    checked?: boolean;
    disabled?: boolean;
    label?: string;
    ariaLabel?: string;
  }

  let {
    checked = $bindable(false),
    disabled = false,
    label = '',
    ariaLabel = '',
  }: Props = $props();
</script>

<label class="ts-root">
  {#if label}<span class="ts-label">{label}</span>{/if}
  <span class="ts-switch">
    <input
      type="checkbox"
      bind:checked
      {disabled}
      aria-label={ariaLabel || label || 'toggle'}
    />
    <span class="ts-track"><span class="ts-knob"></span></span>
  </span>
</label>

<style>
  .ts-root {
    display: inline-flex;
    align-items: center;
    gap: 8px;
    cursor: pointer;
    --wails-draggable: none;
    user-select: none;
    -webkit-user-select: none;
  }
  .ts-label {
    color: var(--text-dim);
    font-size: 13px;
    font-weight: 500;
  }
  .ts-label:empty {
    display: none;
  }
  .ts-switch {
    position: relative;
    display: inline-block;
    width: 36px;
    height: 22px;
    flex: none;
  }
  .ts-switch input {
    position: absolute;
    inset: 0;
    width: 100%;
    height: 100%;
    margin: 0;
    opacity: 0;
    cursor: pointer;
    z-index: 2;
  }
  .ts-track {
    position: absolute;
    inset: 0;
    background: var(--switch-off);
    border-radius: 999px;
    transition: background-color 0.18s ease;
  }
  .ts-knob {
    position: absolute;
    top: 2px;
    left: 2px;
    width: 18px;
    height: 18px;
    border-radius: 50%;
    background: #fff;
    box-shadow: 0 1px 2px rgba(0, 0, 0, 0.3);
    transition: transform 0.18s cubic-bezier(0.4, 0.1, 0.2, 1);
  }
  .ts-switch input:checked + .ts-track {
    background: var(--accent);
  }
  .ts-switch input:checked + .ts-track .ts-knob {
    transform: translateX(14px);
  }
  .ts-switch input:disabled {
    cursor: not-allowed;
  }
  .ts-switch input:disabled + .ts-track {
    opacity: 0.45;
  }
  .ts-switch input:focus-visible + .ts-track {
    box-shadow: 0 0 0 3px var(--focus-ring);
  }
</style>
