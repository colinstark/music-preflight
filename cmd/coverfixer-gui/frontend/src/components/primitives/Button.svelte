<script lang="ts">
  import type { Snippet } from 'svelte';

  interface Props {
    type?: 'button' | 'submit';
    variant?: 'default' | 'primary' | 'cancel';
    size?: 'default' | 'large';
    disabled?: boolean;
    hidden?: boolean;
    onclick?: (e: MouseEvent) => void;
    children?: Snippet;
  }

  let {
    type = 'button',
    variant = 'default',
    size = 'default',
    disabled = false,
    hidden = false,
    onclick,
    children,
  }: Props = $props();
</script>

<button
  {type}
  {disabled}
  {hidden}
  {onclick}
  class="btn"
  class:btn-primary={variant === 'primary'}
  class:btn-cancel={variant === 'cancel'}
  class:btn-large={size === 'large'}
>
  {@render children?.()}
</button>

<style>
  .btn {
    background: var(--panel);
    color: var(--text);
    border: 0.5px solid var(--hairline-strong);
    border-radius: var(--radius-field);
    padding: 6px 16px;
    min-width: 84px;
    font-size: 13px;
    font-weight: 500;
    line-height: 1;
    cursor: pointer;
    --wails-draggable: none;
    transition:
      background-color 0.12s ease,
      transform 0.05s ease;
  }
  .btn:hover:not(:disabled) {
    background: var(--panel-hover);
  }
  .btn:active:not(:disabled) {
    transform: scale(0.985);
  }
  .btn-large {
    padding: 10px 28px;
    font-size: 14px;
  }
  .btn-primary {
    background: var(--accent);
    border-color: transparent;
    color: #fff;
    font-weight: 600;
  }
  .btn-primary:hover:not(:disabled) {
    background: var(--accent-hover);
  }
  .btn-primary:active:not(:disabled) {
    background: var(--accent-pressed);
  }
  .btn:disabled {
    opacity: 0.4;
    cursor: not-allowed;
  }
  .btn-cancel:hover:not(:disabled) {
    color: var(--danger);
    border-color: var(--danger);
    background: var(--danger-bg);
  }
</style>
