<script lang="ts">
  // macOS-style segmented control. `value` binds two-ways. Options may be a
  // comma string ("500,480,320" or "MP3:mp3,AAC:aac"), a string[], or
  // {label,value}[]. Exactly one option is active.
  interface Option {
    label: string;
    value: string;
  }

  interface Props {
    value?: string;
    options?: string | string[] | Option[];
    disabled?: boolean;
  }

  let { value = $bindable(''), options = '', disabled = false }: Props = $props();

  function parse(o: string | string[] | Option[]): Option[] {
    if (Array.isArray(o)) {
      if (o.length === 0) return [];
      if (typeof o[0] === 'object') return o as Option[];
      return (o as string[]).map((s) => ({ label: s, value: s }));
    }
    return o
      .split(',')
      .map((s) => {
        const t = s.trim();
        if (!t) return null;
        const i = t.indexOf(':');
        return i < 0
          ? { label: t, value: t }
          : { label: t.slice(0, i).trim(), value: t.slice(i + 1).trim() };
      })
      .filter((x): x is Option => x !== null);
  }

  const opts = $derived(parse(options));
</script>

<div class="tb" role="tablist">
  {#each opts as o (o.value)}
    <button
      type="button"
      class="tb-tab"
      class:is-active={o.value === value}
      role="tab"
      aria-selected={o.value === value}
      {disabled}
      onclick={() => (value = o.value)}
    >
      {o.label}
    </button>
  {/each}
</div>

<style>
  .tb {
    display: inline-flex;
    gap: 2px;
    padding: 2px;
    background: var(--panel-inset);
    border: 0.5px solid var(--tab-border);
    border-radius: var(--radius-field);
    align-self: flex-start;
    max-width: 100%;
    --wails-draggable: none;
  }
  .tb-tab {
    appearance: none;
    background: transparent;
    border: none;
    color: var(--text-dim);
    font-family: inherit;
    font-size: 12px;
    font-weight: 500;
    padding: 4px 12px;
    border-radius: 4px;
    cursor: pointer;
    white-space: nowrap;
    transition:
      background-color 0.12s ease,
      color 0.12s ease;
  }
  .tb-tab:hover {
    color: var(--text);
  }
  .tb-tab.is-active {
    background: var(--tab-active);
    color: var(--text);
    font-weight: 600;
    box-shadow: 0 0.5px 1.5px rgba(0, 0, 0, 0.25);
  }
  .tb:has(.tb-tab:disabled) {
    opacity: 0.45;
    pointer-events: none;
  }
</style>
