// <toggle-switch> — a reusable macOS-style sliding toggle.
//
// Encapsulates its markup and styles in a shadow root. The host proxies the
// native checkbox API (.checked / .disabled), so it's a drop-in anywhere the
// old <input type="checkbox"> was used. Custom properties (--accent,
// --switch-off, --focus-ring, --text-dim) are defined on :root and inherit
// through the shadow boundary, so light/dark adaptation works for free.
//
// Usage:
//   <toggle-switch label="Dry Run" checked></toggle-switch>
//
// Attributes: checked, disabled, label. Emits a bubbling `change` event.

(function () {
    const template = document.createElement("template");
    template.innerHTML = `
        <label class="ts-root">
            <span class="ts-label"></span>
            <span class="ts-switch">
                <input type="checkbox" />
                <span class="ts-track"><span class="ts-knob"></span></span>
            </span>
        </label>
        <style>
            :host {
                display: inline-flex;
                align-items: center;
                /* Opt out of Wails window-drag and text selection. */
                --wails-draggable: none;
                user-select: none;
                -webkit-user-select: none;
            }
            .ts-root {
                display: inline-flex;
                align-items: center;
                gap: 8px;
                cursor: pointer;
            }
            .ts-label {
                color: var(--text-dim);
                font-size: 13px;
                font-weight: 500;
            }
            .ts-label:empty { display: none; }
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
            input:checked + .ts-track { background: var(--accent); }
            input:checked + .ts-track .ts-knob { transform: translateX(14px); }
            input:disabled { cursor: not-allowed; }
            input:disabled + .ts-track { opacity: 0.45; }
            input:focus-visible + .ts-track { box-shadow: 0 0 0 3px var(--focus-ring); }
        </style>
    `;

    class ToggleSwitch extends HTMLElement {
        static observedAttributes = ["checked", "disabled", "label", "aria-label"];

        constructor() {
            super();
            this.attachShadow({ mode: "open" }).appendChild(template.content.cloneNode(true));
            this._input = this.shadowRoot.querySelector("input");
            this._labelEl = this.shadowRoot.querySelector(".ts-label");

            // Keep the host attribute in sync and re-emit on the host so
            // light-DOM listeners hear a single `change` event.
            this._input.addEventListener("change", () => {
                this.#reflect("checked", this._input.checked);
                this.dispatchEvent(new CustomEvent("change", {
                    bubbles: true,
                    detail: { checked: this._input.checked },
                }));
            });
        }

        attributeChangedCallback(name, _old, value) {
            const present = value != null;
            if (name === "checked") this._input.checked = present;
            else if (name === "disabled") this._input.disabled = present;
            else if (name === "label") this.#renderLabel(value);
            else if (name === "aria-label") this._input.setAttribute("aria-label", value || "toggle");
        }

        get checked() { return this._input.checked; }
        set checked(v) {
            this._input.checked = !!v;
            this.#reflect("checked", this._input.checked);
        }

        get disabled() { return this._input.disabled; }
        set disabled(v) {
            this._input.disabled = !!v;
            this.#reflect("disabled", this._input.disabled);
        }

        get label() { return this.getAttribute("label") || ""; }
        set label(v) {
            if (v == null || v === "") this.removeAttribute("label");
            else this.setAttribute("label", v);
        }

        #renderLabel(value) {
            this._labelEl.textContent = value || "";
            // Only fall back if no explicit aria-label was provided.
            if (!this.hasAttribute("aria-label")) {
                this._input.setAttribute("aria-label", value || "toggle");
            }
        }

        #reflect(name, on) {
            const has = this.hasAttribute(name);
            if (on && !has) this.setAttribute(name, "");
            else if (!on && has) this.removeAttribute(name);
        }
    }

    customElements.define("toggle-switch", ToggleSwitch);
})();
