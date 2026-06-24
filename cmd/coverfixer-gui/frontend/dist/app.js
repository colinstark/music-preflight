// Coverfixer frontend. Uses the Wails-injected globals (window.go.main.App for
// bound Go methods, window.runtime for events) directly, so there is no bundler
// or build step: this file and index.html/style.css are served verbatim.

const App = () => window.go && window.go.main && window.go.main.App;
const runtime = () => window.runtime;

const $ = (id) => document.getElementById(id);

const INPUT_IDS = [
    "backup",
    "embeddedArtSize", "coverJpgSize",
    "transcodeFormat", "transcodeQuality", "genre", "albumArtist",
    "resizeEmbeddedGroup", "coverJpgGroup", "transcodeGroup", "metadataGroup",
];

const ART_SIZES = ["500", "480", "320", "240"];

function intOf(s) {
    const n = parseInt(s, 10);
    return Number.isFinite(n) ? n : 0;
}

function updateRunEnabled() {
    const hasDir = $("dir").value.trim() !== "";
    $("runBtn").disabled = !hasDir || $("runBtn").dataset.running === "true";
}

function setRunning(running) {
    $("runBtn").dataset.running = running ? "true" : "false";
    $("cancelBtn").hidden = !running;
    $("progressBar").hidden = !running;
    // backup is an idle-only control; hide it while running so the actions row
    // stays a single line (progress bar + Run + Cancel).
    $("backup").hidden = running;
    $("folderBtn").disabled = running;
    for (const id of INPUT_IDS) {
        const el = $(id);
        if (el) el.disabled = running;
    }
    updateRunEnabled();
}

// --- Progress bar -----------------------------------------------------------
// Real runs are indeterminate (the engine emits text, not a fraction), so the
// bar just signals activity with a sweeping fill.

function setIndeterminate() {
    const bar = $("progressBar");
    bar.classList.remove("determinate");
    bar.querySelector(".progress-bar-fill").style.width = "";
}

function applyRequest(req) {
    $("dir").value = req.dir || "";
    setFolderDisplay(req.dir);
    $("embeddedArtSize").value = ART_SIZES.includes(String(req.artSize)) ? String(req.artSize) : "500";
    $("coverJpgSize").value = ART_SIZES.includes(String(req.coverJpgSize)) ? String(req.coverJpgSize) : "500";
    $("backup").checked = !!req.backup;
    // Dry-run is permanently off (the toggle is hidden); runs always mutate.
    // Transcode is chosen as Format × Quality tab-bars; the group's master
    // toggle expresses none (off) vs. the selected combination (on).
    const tc = req.transcode && req.transcode !== "none" ? req.transcode : "mp3-320";
    const [fmt, qual] = tc.split("-");
    $("transcodeFormat").value = fmt || "mp3";
    $("transcodeQuality").value = qual || "320";
    $("transcodeGroup").checked = !!(req.transcode && req.transcode !== "none");
    // The metadata group toggle replaces the old "Set genre" checkbox.
    $("metadataGroup").checked = !!req.setGenre;
    // Two independent cover-art groups: embedded resize, and cover.jpg ops.
    $("resizeEmbeddedGroup").checked = !!req.resizeEmbedded;
    $("coverJpgGroup").checked = !!(req.renameStrayJpg || req.resizeCoverJpg || req.extractCover);
    // Don't overwrite a folder-prefilled genre with the empty default.
    if (!$("genre").value) $("genre").value = req.genre || "";
    updateRunEnabled();
}

function collectRequest() {
    const coverJpg = $("coverJpgGroup").checked;
    return {
        dir: $("dir").value,
        artSize: intOf($("embeddedArtSize").value),
        coverJpgSize: intOf($("coverJpgSize").value),
        recursive: true,
        renameStrayJpg: coverJpg,
        resizeCoverJpg: coverJpg,
        extractCover: coverJpg,
        resizeEmbedded: $("resizeEmbeddedGroup").checked,
        transcode: $("transcodeGroup").checked
            ? $("transcodeFormat").value + "-" + $("transcodeQuality").value
            : "none",
        setGenre: $("metadataGroup").checked,
        genre: $("genre").value,
        setAlbumArtist: $("metadataGroup").checked,
        albumArtist: $("albumArtist").value,
        tagEdits: [...stagedEdits.values()],
        backup: $("backup").checked,
        dryRun: false,
    };
}

function resetOutput() {
    $("summary").textContent = "";
    $("error").textContent = "";
}

// --- Library preview (shown when idle) --------------------------------------

// Most recently loaded albums (before any override), so edits to the Album
// Artist / Genre fields or staged per-album edits can re-render the preview
// without re-scanning.
let cachedAlbums = null;

// Staged per-album metadata edits, keyed by index into cachedAlbums. These are
// NOT written to files immediately: they overlay the preview and are sent with
// the next Run (core.TagEdit blocks). Cleared after a Run (now applied) and on
// folder switch (paths would be stale).
const stagedEdits = new Map();

function escapeHtml(s) {
    return s.replace(/[&<>"']/g, (c) => (
        { "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" }[c]
    ));
}

async function loadPreview() {
    const preview = $("preview");
    const library = $("libraryBlock");
    if ($("runBtn").dataset.running === "true") return;
    const dir = $("dir").value;
    if (!dir) {
        cachedAlbums = null;
        library.hidden = true;
        preview.textContent = "";
        $("folderCounts").textContent = "";
        return;
    }
    library.hidden = false;
    preview.innerHTML = '<div class="preview-empty">Reading library…</div>';
    try {
        const albums = await App().ReadLibrary(dir, true);
        if ($("runBtn").dataset.running === "true") return; // a run started meanwhile
        cachedAlbums = albums;
        renderPreview(albums);
    } catch (err) {
        preview.innerHTML = '<div class="preview-empty">Could not read library: ' + escapeHtml(String(err)) + '</div>';
        $("folderCounts").textContent = "";
    }
}

function renderPreview(albums) {
    const preview = $("preview");
    updateFolderCounts(albums);
    if (!albums || albums.length === 0) {
        preview.innerHTML = '<div class="preview-empty">No audio files found.</div>';
        return;
    }
    preview.textContent = "";
    for (let i = 0; i < albums.length; i++) {
        const el = document.createElement("preview-album");
        el.data = prepareAlbum(albums[i], i);
        el.dataset.idx = String(i);
        preview.appendChild(el);
    }
}

// prepareAlbum overlays pending metadata onto an album for display. Precedence
// (highest wins): a staged per-album edit, then the sidebar's global Album
// Artist / Genre (when the Metadata group is on), then the file tags. Per-track
// title / number / artist are overlaid the same way, and a track's artist is
// shown only when it differs from the album's effective artist. `staged` marks
// the album for the header's unsaved-edit badge.
function prepareAlbum(album, idx) {
    const a = Object.assign({}, album);
    const edit = stagedEdits.get(idx);
    let staged = false;
    if (edit) {
        staged = true;
        a.title = edit.album;
        a.artist = edit.albumArtist;
        a.genre = edit.genre;
        a.year = edit.year;
    } else if ($("metadataGroup").checked) {
        const artist = $("albumArtist").value.trim();
        const genre = $("genre").value.trim();
        if (artist) a.artist = artist;
        if (genre) a.genre = genre;
    }
    const albumArtist = a.artist;
    a.tracks = (album.tracks || []).map((t, j) => {
        const te = edit && edit.tracks[j];
        const artist = te ? te.artist : (t.artist || "");
        const shown = artist && artist !== albumArtist ? artist : "";
        return {
            number: te ? te.trackNumber : t.number,
            title: te ? te.title : t.title,
            artist: shown,
            duration: t.duration,
            disc: t.disc,
        };
    });
    a.staged = staged;
    return a;
}

// Re-render the cached preview (no re-fetch) to reflect metadata edits live.
let refreshTimer = null;
function refreshPreviewDisplay() {
    if (!cachedAlbums) return;
    clearTimeout(refreshTimer);
    refreshTimer = setTimeout(() => renderPreview(cachedAlbums), 150);
}

// Refresh the preview after a run completes (runs always mutate now). Staged
// edits are cleared: a successful Run wrote them to the files, so the rescan
// picks them up from disk.
function refreshPreviewAfterRun() {
    stagedEdits.clear();
    loadPreview();
}

// --- Per-album edit modal ---------------------------------------------------

let editingIdx = null;     // index into cachedAlbums currently being edited
let editingPaths = null;   // parallel array of track file paths for the album

function openEditModal(idx) {
    if (!cachedAlbums || idx < 0 || idx >= cachedAlbums.length) return;
    const album = cachedAlbums[idx];
    const edit = stagedEdits.get(idx);
    editingIdx = idx;
    editingPaths = (album.tracks || []).map((t) => t.path || "");

    $("editHeading").textContent = "Edit — " + (album.title || "Unknown Album");
    $("editAlbumTitle").value = edit ? edit.album : (album.title || "");
    $("editAlbumArtist").value = edit ? edit.albumArtist : (album.artist || "");
    $("editGenre").value = edit ? edit.genre : (album.genre || "");
    $("editYear").value = edit ? edit.year : (album.year || "");
    $("editStatus").textContent = edit ? "Unsaved edits staged" : "";
    $("editStatus").classList.toggle("ok", false);

    const list = $("editTrackList");
    list.textContent = "";
    (album.tracks || []).forEach((t, j) => {
        const te = edit && edit.tracks[j];
        const row = document.createElement("label");
        row.className = "edit-track";
        const num = document.createElement("input");
        num.type = "number"; num.min = "0"; num.className = "et-num";
        num.value = String(te ? te.trackNumber : (t.number || ""));
        const title = document.createElement("input");
        title.type = "text"; title.className = "et-title";
        title.value = te ? te.title : (t.title || "");
        const artist = document.createElement("input");
        artist.type = "text"; artist.className = "et-artist";
        artist.value = te ? te.artist : (t.artist || "");
        artist.placeholder = "(album artist)";
        row.append(num, title, artist);
        list.appendChild(row);
    });

    $("editModal").showModal();
    $("editAlbumTitle").focus();
    $("editAlbumTitle").select();
}

function closeEditModal() {
    editingIdx = null;
    editingPaths = null;
    const dlg = $("editModal");
    if (dlg.open) dlg.close();
}

// Gather the form into a core.TagEdit-shaped object.
function gatherEditFromForm() {
    const rows = $("editTrackList").querySelectorAll(".edit-track");
    const tracks = [];
    rows.forEach((row, j) => {
        tracks.push({
            path: editingPaths[j] || "",
            title: row.querySelector(".et-title").value,
            artist: row.querySelector(".et-artist").value,
            trackNumber: intOf(row.querySelector(".et-num").value),
        });
    });
    return {
        album: $("editAlbumTitle").value,
        albumArtist: $("editAlbumArtist").value,
        genre: $("editGenre").value,
        year: $("editYear").value,
        tracks,
    };
}

// editFromAlbum builds the same shape from an album's on-disk tags — the
// baseline against which a Save is compared so an unchanged Save stages nothing.
function editFromAlbum(album) {
    return {
        album: album.title || "",
        albumArtist: album.artist || "",
        genre: album.genre || "",
        year: album.year || "",
        tracks: (album.tracks || []).map((t) => ({
            path: t.path || "",
            title: t.title || "",
            artist: t.artist || "",
            trackNumber: t.number || 0,
        })),
    };
}

function editsEqual(a, b) {
    if (a.album !== b.album || a.albumArtist !== b.albumArtist ||
        a.genre !== b.genre || a.year !== b.year) {
        return false;
    }
    const at = a.tracks || [], bt = b.tracks || [];
    if (at.length !== bt.length) return false;
    for (let i = 0; i < at.length; i++) {
        if (at[i].title !== bt[i].title || at[i].artist !== bt[i].artist ||
            at[i].trackNumber !== bt[i].trackNumber) {
            return false;
        }
    }
    return true;
}

// Stage the form for the next Run — but only if it actually differs from the
// files' current tags. An unchanged Save (or one reverted to the baseline)
// unstages the album so it shows no badge and isn't sent on Run.
function stageEditFromForm() {
    if (editingIdx === null || !cachedAlbums) return;
    const desired = gatherEditFromForm();
    const baseline = editFromAlbum(cachedAlbums[editingIdx]);
    if (editsEqual(desired, baseline)) {
        stagedEdits.delete(editingIdx);
    } else {
        stagedEdits.set(editingIdx, desired);
    }
    refreshPreviewDisplay();
}

// selectFolder is the single entry point for setting the selected folder,
// shared by the picker, a window drop, and launch-by-drop. It mirrors what the
// picker used to do inline: set the path, prefill metadata from the first audio
// file, then refresh the run-enable state and preview. A no-op while a run is in
// flight (folder changes mid-run would race the preview and the engine).
// folderName returns the last path segment of dir (the folder's display name),
// tolerating both / and \ separators.
function folderName(dir) {
    if (!dir) return "";
    const parts = dir.split(/[\\/]/).filter(Boolean);
    return parts[parts.length - 1] || dir;
}

// setFolderDisplay updates the folder bar: the folder name (not the full path,
// which is kept as a hover tooltip), the folder icon, and clears the counts when
// there is no folder. It also toggles the empty-state view (a single centred
// Choose Folder button) versus the full folder bar + content.
function setFolderDisplay(dir) {
    const label = $("pathLabel");
    const icon = $("folderIcon");
    const hasDir = !!dir;
    $("emptyState").hidden = hasDir;
    document.querySelector(".folder-row").hidden = !hasDir;
    document.querySelector(".content").hidden = !hasDir;
    if (hasDir) {
        label.textContent = folderName(dir);
        label.title = dir;
        if (icon) icon.hidden = false;
    } else {
        label.textContent = "No folder selected";
        label.title = "";
        if (icon) icon.hidden = true;
        $("folderCounts").textContent = "";
    }
}

// updateFolderCounts writes "· N Albums, M Tracks" (singular-aware) from the
// loaded albums, or clears it when there are none.
function updateFolderCounts(albums) {
    const el = $("folderCounts");
    const aN = albums ? albums.length : 0;
    if (aN === 0) {
        el.textContent = "";
        return;
    }
    let tN = 0;
    for (const a of albums) tN += (a.tracks || []).length;
    el.textContent =
        "· " + aN + " Album" + (aN === 1 ? "" : "s") +
        ", " + tN + " Track" + (tN === 1 ? "" : "s");
}

async function selectFolder(dir) {
    if (!dir) return;
    if ($("runBtn").dataset.running === "true") return;
    stagedEdits.clear();
    $("dir").value = dir;
    setFolderDisplay(dir);
    // Prefill the metadata fields from the first audio file's existing tags.
    try {
        const m = await App().ReadFirstMetadata(dir);
        if ($("runBtn").dataset.running === "true") return; // a run started meanwhile
        $("albumArtist").value = m.albumArtist || "";
        $("genre").value = m.genre || "";
    } catch (err) {
        console.error(err);
    }
    updateRunEnabled();
    loadPreview();
}

async function onChooseFolder() {
    const dir = await App().OpenFolder();
    if (dir) await selectFolder(dir);
}

async function onRun() {
    if ($("runBtn").dataset.running === "true") return;
    if (!$("dir").value) return;
    resetOutput();
    setIndeterminate();
    setRunning(true);
    try {
        // Run returns once the engine has started; progress and completion
        // arrive via the cf:* events. A rejected promise here means the run
        // never started (already running, or an invalid request).
        await App().Run(collectRequest());
    } catch (err) {
        $("error").textContent = String(err);
        setRunning(false);
    }
}

function onCancel() {
    App().Cancel();
}

function onDone(summary) {
    $("summary").textContent = summary || "";
    setRunning(false);
    refreshPreviewAfterRun();
}

function onError(msg) {
    $("error").textContent = msg || "";
    setRunning(false);
    // Keep staged edits on failure so the user can retry without re-entering
    // them; the rescan reflects whatever the engine actually wrote.
    loadPreview();
}

async function init() {
    // Wait for Wails to inject the runtime + bindings (normally synchronous,
    // but guard against a startup race).
    const t0 = Date.now();
    while (!(App() && runtime())) {
        if (Date.now() - t0 > 5000) {
            $("error").textContent = "Failed to connect to the backend runtime.";
            return;
        }
        await new Promise((r) => setTimeout(r, 50));
    }

    // Seed the form from the single source of truth in Go.
    try {
        applyRequest(await App().DefaultRequest());
    } catch (err) {
        // Non-fatal: leave the form at its HTML defaults.
        console.error(err);
    }

    runtime().EventsOn("cf:done", onDone);
    runtime().EventsOn("cf:error", onError);
    runtime().EventsOn("cf:state", (running) => setRunning(!!running));
    // A folder dropped on the window or the Dock icon while the app is running.
    // (A launch-by-drop is pulled separately via InitialFolder below, since that
    // event can fire before this listener is registered.)
    runtime().EventsOn("cf:folder", (dir) => { if (dir) selectFolder(dir); });

    $("folderBtn").addEventListener("click", onChooseFolder);
    $("emptyFolderBtn").addEventListener("click", onChooseFolder);
    $("runBtn").addEventListener("click", onRun);
    $("cancelBtn").addEventListener("click", onCancel);
    // Reflect pending Album Artist / Genre edits in the preview live.
    $("albumArtist").addEventListener("input", refreshPreviewDisplay);
    $("genre").addEventListener("input", refreshPreviewDisplay);
    $("metadataGroup").addEventListener("change", refreshPreviewDisplay);

    // Per-album edit modal. The album-edit event crosses the preview-album /
    // album-header shadow boundaries composed, retargeted to the host whose
    // data-idx names the album.
    $("preview").addEventListener("album-edit", (e) => {
        const idx = parseInt(e.target.dataset.idx, 10);
        if (Number.isInteger(idx)) openEditModal(idx);
    });
    // method="dialog" submits close the dialog; stage the edit on submit.
    $("editForm").addEventListener("submit", stageEditFromForm);
    $("editCancel").addEventListener("click", closeEditModal);
    // Esc on the dialog: nothing to undo (no tentative DOM), just reset state.
    $("editModal").addEventListener("close", () => { editingIdx = null; editingPaths = null; });
    $("editRevert").addEventListener("click", () => {
        if (editingIdx !== null) {
            stagedEdits.delete(editingIdx);
            refreshPreviewDisplay();
        }
        closeEditModal();
    });

    // Apply a folder passed at launch (e.g. dropped on the Dock icon before the
    // app was running), then paint the idle empty/loaded state.
    try {
        const initial = await App().InitialFolder();
        if (initial) {
            await selectFolder(initial);
        } else {
            loadPreview();
        }
    } catch (err) {
        console.error(err);
        loadPreview();
    }
}

document.addEventListener("DOMContentLoaded", init);
