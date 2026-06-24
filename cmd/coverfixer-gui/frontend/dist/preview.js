// Library preview web components.
//
// Three composable custom elements, each with its own shadow DOM:
//   <album-track n title duration>   — "N. Title (M:SS)"
//   <album-header title artist genre artwork> — art thumb + stacked meta
//   <preview-album>                  — composes a header + track list from a
//                                       .data album object (set in JS)
//
// Theme tokens (--panel, --text, --text-dim, --hairline, --radius, etc.) are
// defined on :root in style.css and inherit through every shadow boundary, so
// light/dark adaptation works for free.

(function () {
    const PALETTE_STYLE = `
        :host {
            display: block;
            --wails-draggable: none;
        }
    `;

    function formatDuration(sec) {
        if (!sec || sec <= 0) return "";
        sec = Math.round(sec);
        const m = Math.floor(sec / 60);
        const s = sec % 60;
        return m + ":" + String(s).padStart(2, "0");
    }

    // discGroups groups tracks by disc number, folding an unset disc (0) to 1,
    // and returns the groups sorted by disc. A single-disc album yields one
    // group so the caller can skip the "Disc N" heading.
    function discGroups(tracks) {
        const map = new Map();
        for (const t of tracks) {
            const d = (t.disc && t.disc > 0) ? t.disc : 1;
            if (!map.has(d)) map.set(d, []);
            map.get(d).push(t);
        }
        return [...map.entries()]
            .sort((a, b) => a[0] - b[0])
            .map(([disc, tr]) => ({ disc, tracks: tr }));
    }

    // <album-track n title artist duration>
    class AlbumTrack extends HTMLElement {
        static observedAttributes = ["n", "name", "artist", "duration"];

        constructor() {
            super();
            const root = this.attachShadow({ mode: "open" });
            root.innerHTML = `
                <li class="at">
                    <span class="at-n"></span>
                    <span class="at-title"></span>
                    <span class="at-dur"></span>
                </li>
                <style>
                    :host { display: list-item; }
                    .at {
                        display: grid;
                        grid-template-columns: 2ch 1fr auto;
                        column-gap: 8px;
                        align-items: baseline;
                        padding: 1px 0 1px 12px;
                        margin-bottom: 4px;
                    }
                    .at-n {
                        color: var(--text-faint);
                        font-variant-numeric: tabular-nums;
                        text-align: right;
                        font-size: 12px;
                    }
                    .at-title {
                        color: var(--text-dim);
                        overflow: hidden;
                        text-overflow: ellipsis;
                        white-space: nowrap;
                        font-size: 12.5px;
                    }
                    .at-artist {
                        color: var(--text-faint);
                        font-size: 12.5px;
                    }
                    .at-artist:empty { display: none; }
                    .at-dur {
                        color: var(--text-faint);
                        font-variant-numeric: tabular-nums;
                        font-size: 12px;
                    }
                </style>
            `;
            this._nEl = root.querySelector(".at-n");
            this._titleEl = root.querySelector(".at-title");
            this._durEl = root.querySelector(".at-dur");
        }

        attributeChangedCallback() { this._render(); }
        connectedCallback() { this._render(); }

        _render() {
            const n = this.getAttribute("n");
            this._nEl.textContent = n ? n + "." : "";
            const artist = this.getAttribute("artist") || "";
            // Title and (optional) per-track artist share the title cell as
            // "Title — Artist"; the artist span is hidden when empty.
            this._titleEl.innerHTML = "";
            this._titleEl.appendChild(document.createTextNode(this.getAttribute("name") || ""));
            if (artist) {
                const sep = document.createElement("span");
                sep.className = "at-artist";
                sep.textContent = " — " + artist;
                this._titleEl.appendChild(sep);
            }
            this._durEl.textContent = formatDuration(parseFloat(this.getAttribute("duration") || "0"));
        }
    }

    // <album-header title artist genre year artwork staged>
    class AlbumHeader extends HTMLElement {
        static observedAttributes = ["heading", "artist", "genre", "year", "artwork", "staged"];

        constructor() {
            super();
            const root = this.attachShadow({ mode: "open" });
            root.innerHTML = `
                <div class="ah">
                    <div class="ah-art">
                        <img alt="" />
                        <div class="ah-art-ph"></div>
                        <span class="ah-badge" title="Has unsaved edits"></span>
                    </div>
                    <div class="ah-meta">
                        <div class="ah-title"></div>
                        <div class="ah-artist"></div>
                        <div class="ah-genre"></div>
                    </div>
                    <button class="ah-edit" type="button" title="Edit metadata">Edit</button>
                </div>
                <style>
                    .ah {
                        position: relative;
                        display: flex;
                        align-items: center;
                        gap: 12px;
                        margin-bottom: 8px;
                    }
                    .ah-art {
                        position: relative;
                        width: 44px;
                        height: 44px;
                        flex: none;
                        border-radius: 5px;
                        overflow: hidden;
                        background: var(--panel-inset);
                        border: 0.5px solid var(--hairline);
                    }
                    .ah-art img {
                        width: 100%;
                        height: 100%;
                        object-fit: cover;
                        display: block;
                    }
                    .ah-art img:not([src]) { display: none; }
                    .ah-badge {
                        position: absolute;
                        top: 3px;
                        right: 3px;
                        width: 8px;
                        height: 8px;
                        border-radius: 50%;
                        background: var(--accent);
                        border: 1.5px solid var(--panel);
                        box-shadow: 0 0 0 1px var(--hairline);
                        display: none;
                    }
                    :host([staged]) .ah-badge { display: block; }
                    .ah-title {
                        color: var(--text);
                        font-size: 13px;
                        font-weight: 600;
                        overflow: hidden;
                        text-overflow: ellipsis;
                        white-space: nowrap;
                    }
                    .ah-artist {
                        color: var(--text-dim);
                        font-size: 12px;
                        margin-top: 1px;
                        overflow: hidden;
                        text-overflow: ellipsis;
                        white-space: nowrap;
                    }
                    .ah-genre {
                        color: var(--text-faint);
                        font-size: 11px;
                        margin-top: 1px;
                    }
                    .ah-genre:empty { display: none; }
                    .ah-edit {
                        position: absolute;
                        top: -3px;
                        right: -3px;
                        display: flex;
                        align-items: center;
                        justify-content: center;
                        height: 20px;
                        padding: 0 8px;
                        border: 0.5px solid var(--hairline);
                        border-radius: 5px;
                        background: var(--panel);
                        color: var(--text-dim);
                        font-size: 11px;
                        font-weight: 500;
                        cursor: pointer;
                        opacity: 0;
                        transition: opacity 0.12s ease;
                    }
                    .ah:hover .ah-edit,
                    .ah-edit:focus-visible { opacity: 1; }
                    .ah-edit:hover { color: var(--text); border-color: var(--accent); }
                </style>
            `;
            this._img = root.querySelector("img");
            this._titleEl = root.querySelector(".ah-title");
            this._artistEl = root.querySelector(".ah-artist");
            this._genreEl = root.querySelector(".ah-genre");
            // The edit button crosses two shadow boundaries (album-header's and
            // preview-album's); composed+bubbles lets app.js hear it retargeted
            // to the <preview-album> host, whose data-idx names the album.
            root.querySelector(".ah-edit").addEventListener("click", (e) => {
                e.preventDefault();
                e.stopPropagation();
                this.dispatchEvent(new CustomEvent("album-edit", { bubbles: true, composed: true }));
            });
        }

        attributeChangedCallback() { this._render(); }
        connectedCallback() { this._render(); }

        _render() {
            this._titleEl.textContent = this.getAttribute("heading") || "Unknown Album";
            this._artistEl.textContent = this.getAttribute("artist") || "";
            // Genre and year share one line: "genre · year" (whichever exist).
            const genre = this.getAttribute("genre");
            const year = this.getAttribute("year");
            this._genreEl.textContent = [genre, year].filter(Boolean).join(" · ");
            const src = this.getAttribute("artwork");
            if (src) this._img.src = src;
            else this._img.removeAttribute("src");
        }
    }

    // <preview-album> — composes a header + track list from a .data object.
    class PreviewAlbum extends HTMLElement {
        constructor() {
            super();
            const root = this.attachShadow({ mode: "open" });
            root.innerHTML = `
                <section class="pa">
                    <album-header></album-header>
                    <div class="pa-body"></div>
                </section>
                <style>
                    ${PALETTE_STYLE}
                    .pa {
                        padding: 10px 12px;
                        background: var(--panel);
                        border: 0.5px solid var(--hairline);
                        border-radius: var(--radius);
                    }
                    .pa-ol {
                        margin: 0;
                        padding-left: 22px;
                        list-style: none;
                    }
                    .pa-disc {
                        color: var(--text-dim);
                        font-size: 11px;
                        font-weight: 600;
                        text-transform: uppercase;
                        letter-spacing: 0.04em;
                        margin: 8px 0 2px;
                        padding: 4px 0 4px calc(44px + 2ch);
                    }
                    .pa-body > .pa-disc:first-child { margin-top: 0; }
                </style>
            `;
            this._header = root.querySelector("album-header");
            this._body = root.querySelector(".pa-body");
        }

        set data(album) {
            if (!album) return;
            this._header.setAttribute("heading", album.title || "");
            this._header.setAttribute("artist", album.artist || "");
            this._header.setAttribute("genre", album.genre || "");
            this._header.setAttribute("year", album.year || "");
            if (album.artwork) this._header.setAttribute("artwork", album.artwork);
            else this._header.removeAttribute("artwork");
            if (album.staged) this._header.setAttribute("staged", "");
            else this._header.removeAttribute("staged");

            // Split tracks into disc groups; only render "Disc N" headings when
            // the album spans more than one disc. Tracks arrive sorted by the
            // backend (disc, number, title).
            const body = this._body;
            body.textContent = "";
            const groups = discGroups(album.tracks || []);
            for (const g of groups) {
                if (groups.length > 1) {
                    const h = document.createElement("div");
                    h.className = "pa-disc";
                    h.textContent = "Disc " + g.disc;
                    body.appendChild(h);
                }
                const ol = document.createElement("ol");
                ol.className = "pa-ol";
                for (const t of g.tracks) {
                    const tr = document.createElement("album-track");
                    if (t.number) tr.setAttribute("n", String(t.number));
                    tr.setAttribute("name", t.title || "");
                    if (t.artist) tr.setAttribute("artist", t.artist);
                    tr.setAttribute("duration", String(t.duration || 0));
                    ol.appendChild(tr);
                }
                body.appendChild(ol);
            }
        }
    }

    customElements.define("album-track", AlbumTrack);
    customElements.define("album-header", AlbumHeader);
    customElements.define("preview-album", PreviewAlbum);
})();
