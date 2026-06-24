<script lang="ts">
  // A labelled text/number input. bind:value two-ways with the caller's state.
  import type { HTMLInputAttributes } from 'svelte/elements';

  interface Props {
    value?: string;
    placeholder?: string;
    disabled?: boolean;
    autocomplete?: HTMLInputAttributes['autocomplete'];
    inputmode?: HTMLInputAttributes['inputmode'];
    type?: HTMLInputAttributes['type'];
    id?: string;
  }

  let {
    value = $bindable(''),
    placeholder = '',
    disabled = false,
    autocomplete = 'off',
    inputmode = undefined,
    type = 'text',
    id = undefined,
  }: Props = $props();
</script>

<input {type} {placeholder} {disabled} {autocomplete} {inputmode} {id} bind:value />

<style>
  input {
    background: var(--field-bg);
    color: var(--text);
    border: 0.5px solid var(--hairline-strong);
    border-radius: var(--radius-field);
    padding: 4px 8px;
    font-size: 13px;
    font-family: inherit;
    width: 100%;
    min-width: 0;
    --wails-draggable: none;
    accent-color: var(--accent);
  }
  input::placeholder {
    color: var(--text-faint);
  }
  input:focus {
    outline: none;
    border-color: var(--accent);
    box-shadow: 0 0 0 3px var(--focus-ring);
  }
  input:disabled {
    opacity: 0.55;
  }
</style>
