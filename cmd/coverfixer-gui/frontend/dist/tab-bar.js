// <tab-bar> — a reusable macOS-style segmented control.
//
// Renders one button per option in a recessed track; exactly one is active.
// The active option's value is exposed via the `.value` property/attribute and
// a bubbling `change` event (detail.value) fires on selection.
//
// Usage:
//   <tab-bar value="480" options="600,480,320,240"></tab-bar>
//   <tab-bar value="mp3" options="MP3:mp3,AAC:aac"></tab-bar>   <!-- label:value -->
//
// Options is a comma-separated list; each item is "value" (label==value) or
// "label:value". Theme tokens inherit through the shadow boundary.

(function () {
    class TabBar extends HTMLElement {
        static observedAttributes = ["value", "options"];

        constructor() {
            super();
            const root = this.attachShadow({ mode: "open" });
            // Event delegation inside the shadow root so re-renders need no rebinding.
            root.addEventListener("click", (e) => {
                const btn = e.target.closest("button.tb-tab");
                if (!btn || this.disabled) return;
                const v = btn.dataset.value;
                if (v === this.value) return;
                this.value = v;
                this.dispatchEvent(new CustomEvent("change", { bubbles: true, detail: { value: v } }));
            });
        }

        connectedCallback() { this._render(); }
        attributeChangedCallback() { this._render(); }

        get value() { return this.getAttribute("value") || ""; }
        set value(v) { this.setAttribute("value", v); }

        get options() { return this.getAttribute("options") || ""; }
        set options(v) { this.setAttribute("options", v); }

        get disabled() { return this.hasAttribute("disabled"); }
        set disabled(v) {
            if (v) this.setAttribute("disabled", "");
            else this.removeAttribute("disabled");
        }

        _parseOptions() {
            return this.options.split(",").map((s) => {
                const t = s.trim();
                if (!t) return null;
                const i = t.indexOf(":");
                return i < 0 ? { label: t, value: t } : { label: t.slice(0, i).trim(), value: t.slice(i + 1).trim() };
            }).filter(Boolean);
        }

        _render() {
            const opts = this._parseOptions();
            const cur = this.value;
            const tabs = opts.map((o) => {
                const active = o.value === cur;
                return `<button type="button" class="tb-tab${active ? " is-active" : ""}" data-value="${o.value}" role="tab" aria-selected="${active}">${o.label}</button>`;
            }).join("");
            this.shadowRoot.innerHTML = `
                <div class="tb" role="tablist">${tabs}</div>
                <style>
                    :host {
                        display: inline-flex;
                        --wails-draggable: none;
                    }
                    .tb {
                        display: inline-flex;
                        gap: 2px;
                        padding: 2px;
                        background: var(--panel-inset);
                        border: 0.5px solid var(--tab-border);
                        border-radius: var(--radius-field);
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
                        transition: background-color 0.12s ease, color 0.12s ease;
                    }
                    .tb-tab:hover { color: var(--text); }
                    .tb-tab.is-active {
                        background: var(--tab-active);
                        color: var(--text);
                        font-weight: 600;
                        box-shadow: 0 0.5px 1.5px rgba(0, 0, 0, 0.25);
                    }
                    :host([disabled]) .tb { opacity: 0.45; pointer-events: none; }
                </style>
            `;
        }
    }

    customElements.define("tab-bar", TabBar);
})();
