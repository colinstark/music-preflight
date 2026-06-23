// <option-group> — a labelled container that structures a set of related options.
//
// Renders a full-width header (headline on the left, a master <toggle-switch>
// on the right) above a slotted content area. The master toggle is the group's
// "enabled" flag: when off the content is collapsed (hidden) and callers should
// treat the group's options as inactive (see app.js collectRequest). Turning it
// on expands the content with a smooth auto-height animation.
//
// Usage:
//   <option-group id="..." label="Cover art" checked>
//       <!-- controls -->
//   </option-group>
//
// Attributes: label, checked (presence = on; default off when absent),
// disabled (locks the master toggle, e.g. during a run).
// Exposes .checked / .disabled / .label. Emits a bubbling `change` event.

(function () {
    const template = document.createElement("template");
    template.innerHTML = `
        <div class="og">
            <div class="og-header">
                <span class="og-label"></span>
                <toggle-switch></toggle-switch>
            </div>
            <div class="og-content-wrap"><div class="og-content"><slot></slot></div></div>
        </div>
        <style>
            :host {
                display: block;
                /* Symmetric vertical padding so a closed group has the same
                   space below its header as above. */
                padding: 12px 14px;
                /* Not a window drag handle; not selectable. */
                --wails-draggable: none;
                user-select: none;
                -webkit-user-select: none;
            }
            .og-header {
                display: flex;
                align-items: center;
                justify-content: space-between;
                gap: 10px;
            }
            .og-label {
                font-size: 13px;
                font-weight: 600;
                color: var(--text);
            }
            /* Collapse/expand via max-height on the wrapper (overflow hidden
               guarantees a closed group is truly 0 tall — no leaked padding).
               Height uses easeOutQuad; the content fades in shortly after. */
            .og-content-wrap {
                overflow: hidden;
                max-height: 0;
                transition: max-height 0.22s cubic-bezier(0.25, 0.46, 0.45, 0.94);
            }
            :host([checked]) .og-content-wrap { max-height: 320px; }
            .og-content {
                padding-top: 10px;
                padding-bottom: 0;
                opacity: 0;
                transition: opacity 0.15s ease;
            }
            :host([checked]) .og-content {
                opacity: 1;
                transition-delay: 0.08s;
            }
            :host(:not([checked])) .og-content { pointer-events: none; }
        </style>
    `;

    class OptionGroup extends HTMLElement {
        static observedAttributes = ["checked", "label", "disabled"];

        constructor() {
            super();
            this.attachShadow({ mode: "open" }).appendChild(template.content.cloneNode(true));
            this._toggle = this.shadowRoot.querySelector("toggle-switch");
            this._labelEl = this.shadowRoot.querySelector(".og-label");

            this._toggle.addEventListener("change", () => {
                const on = this._toggle.checked;
                if (on) this.setAttribute("checked", "");
                else this.removeAttribute("checked");
                this.dispatchEvent(new CustomEvent("change", {
                    bubbles: true,
                    detail: { checked: on },
                }));
            });
        }

        attributeChangedCallback(name, _old, value) {
            if (name === "checked") {
                this._toggle.checked = value != null;
            } else if (name === "label") {
                this._labelEl.textContent = value || "";
                this._toggle.setAttribute("aria-label", value ? `${value} (master enable)` : "option group");
            } else if (name === "disabled") {
                this._toggle.disabled = value != null;
            }
        }

        get checked() { return this.hasAttribute("checked"); }
        set checked(v) {
            if (v) this.setAttribute("checked", "");
            else this.removeAttribute("checked");
        }

        get disabled() { return this._toggle.disabled; }
        set disabled(v) { this._toggle.disabled = !!v; }

        get label() { return this.getAttribute("label") || ""; }
        set label(v) {
            if (v == null || v === "") this.removeAttribute("label");
            else this.setAttribute("label", v);
        }
    }

    customElements.define("option-group", OptionGroup);
})();
