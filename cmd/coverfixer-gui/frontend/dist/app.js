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
    // These are idle-only controls; hide them while running so the actions
    // row stays a single line (progress bar + Run + Cancel).
    $("backup").hidden = running;
    $("debugRunBtn").hidden = running;
    $("folderBtn").disabled = running;
    for (const id of INPUT_IDS) {
        const el = $(id);
        if (el) el.disabled = running;
    }
    updateRunEnabled();
}

// --- Progress bar -----------------------------------------------------------
// Real runs are indeterminate (the engine emits text, not a fraction). The
// debug run drives a determinate 0..100% fill. setProgress toggles determinate
// mode and sizes the fill; setIndeterminate restores the sweeping animation.

function setProgress(fraction) {
    const bar = $("progressBar");
    bar.classList.add("determinate");
    bar.querySelector(".progress-bar-fill").style.width =
        Math.max(0, Math.min(1, fraction)) * 100 + "%";
}

function setIndeterminate() {
    const bar = $("progressBar");
    bar.classList.remove("determinate");
    bar.querySelector(".progress-bar-fill").style.width = "";
}

function applyRequest(req) {
    $("dir").value = req.dir || "";
    $("pathLabel").textContent = req.dir || "No folder selected";
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
        backup: $("backup").checked,
        dryRun: false,
    };
}

function resetOutput() {
    $("summary").textContent = "";
    $("error").textContent = "";
}

// --- Library preview (shown when idle) --------------------------------------

// Most recently loaded albums (before any metadata override), so edits to the
// Album Artist / Genre fields can re-render the preview without re-scanning.
let cachedAlbums = null;

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
    }
}

function renderPreview(albums) {
    const preview = $("preview");
    if (!albums || albums.length === 0) {
        preview.innerHTML = '<div class="preview-empty">No audio files found.</div>';
        return;
    }
    preview.textContent = "";
    for (const album of albums) {
        const el = document.createElement("preview-album");
        el.data = metadataOverride(album);
        preview.appendChild(el);
    }
}

// When the Metadata group is enabled, the Album Artist / Genre field values are
// written to every file on Run; reflect that pending state in the preview by
// overriding each album's displayed artist/genre with the (non-empty) field
// values. The original grouping is preserved.
function metadataOverride(album) {
    if (!$("metadataGroup").checked) return album;
    const a = Object.assign({}, album);
    const artist = $("albumArtist").value.trim();
    const genre = $("genre").value.trim();
    if (artist) a.artist = artist;
    if (genre) a.genre = genre;
    return a;
}

// Re-render the cached preview (no re-fetch) to reflect metadata edits live.
let refreshTimer = null;
function refreshPreviewDisplay() {
    if (!cachedAlbums) return;
    clearTimeout(refreshTimer);
    refreshTimer = setTimeout(() => renderPreview(cachedAlbums), 150);
}

// Refresh the preview after a run completes (runs always mutate now).
function refreshPreviewAfterRun() {
    loadPreview();
}

async function onChooseFolder() {
    const dir = await App().OpenFolder();
    if (dir) {
        $("dir").value = dir;
        $("pathLabel").textContent = dir;
        // Prefill the metadata fields from the first audio file's existing tags.
        try {
            const m = await App().ReadFirstMetadata(dir);
            $("albumArtist").value = m.albumArtist || "";
            $("genre").value = m.genre || "";
        } catch (err) {
            console.error(err);
        }
        updateRunEnabled();
        loadPreview();
    }
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

// onDebugRun mocks a run for UI debugging: it enters the running state and
// advances the progress bar 20% per second to 100%, without touching the
// engine. Cancel stops it early.
let debugTimer = null;
function onDebugRun() {
    if ($("runBtn").dataset.running === "true") return;
    resetOutput();
    setRunning(true);
    let pct = 0;
    setProgress(0);
    debugTimer = setInterval(() => {
        pct = Math.min(100, pct + 20);
        setProgress(pct / 100);
        if (pct >= 100) {
            clearInterval(debugTimer);
            debugTimer = null;
            $("summary").textContent = "Debug run complete.";
            setRunning(false);
        }
    }, 1000);
}

function onCancel() {
    // A debug run has no engine to cancel: just stop its timer.
    if (debugTimer) {
        clearInterval(debugTimer);
        debugTimer = null;
        setRunning(false);
        return;
    }
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
    refreshPreviewAfterRun();
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

    $("folderBtn").addEventListener("click", onChooseFolder);
    $("runBtn").addEventListener("click", onRun);
    $("debugRunBtn").addEventListener("click", onDebugRun);
    $("cancelBtn").addEventListener("click", onCancel);
    // Reflect pending Album Artist / Genre edits in the preview live.
    $("albumArtist").addEventListener("input", refreshPreviewDisplay);
    $("genre").addEventListener("input", refreshPreviewDisplay);
    $("metadataGroup").addEventListener("change", refreshPreviewDisplay);

    loadPreview(); // paint the idle empty state
}

document.addEventListener("DOMContentLoaded", init);
