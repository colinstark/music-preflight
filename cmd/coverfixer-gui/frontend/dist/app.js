// Coverfixer frontend. Uses the Wails-injected globals (window.go.main.App for
// bound Go methods, window.runtime for events) directly, so there is no bundler
// or build step: this file and index.html/style.css are served verbatim.

const App = () => window.go && window.go.main && window.go.main.App;
const runtime = () => window.runtime;

const $ = (id) => document.getElementById(id);

const INPUT_IDS = [
    "dryRun", "recursive", "renameStrayJpg", "resizeCoverJpg",
    "extractCover", "resizeEmbedded", "backup",
    "artSize", "jpegQuality", "transcode",
];

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
    $("cancelBtn").disabled = !running;
    $("folderBtn").disabled = running;
    for (const id of INPUT_IDS) {
        const el = $(id);
        if (el) el.disabled = running;
    }
    updateRunEnabled();
}

function applyRequest(req) {
    $("dir").value = req.dir || "";
    $("pathLabel").textContent = req.dir || "No folder selected";
    $("artSize").value = req.artSize || "";
    $("jpegQuality").value = req.jpegQuality || "";
    $("recursive").checked = !!req.recursive;
    $("renameStrayJpg").checked = !!req.renameStrayJpg;
    $("resizeCoverJpg").checked = !!req.resizeCoverJpg;
    $("extractCover").checked = !!req.extractCover;
    $("resizeEmbedded").checked = !!req.resizeEmbedded;
    $("backup").checked = !!req.backup;
    $("dryRun").checked = !!req.dryRun;
    $("transcode").value = req.transcode || "none";
    updateRunEnabled();
}

function collectRequest() {
    return {
        dir: $("dir").value,
        artSize: intOf($("artSize").value),
        jpegQuality: intOf($("jpegQuality").value),
        recursive: $("recursive").checked,
        renameStrayJpg: $("renameStrayJpg").checked,
        resizeCoverJpg: $("resizeCoverJpg").checked,
        extractCover: $("extractCover").checked,
        resizeEmbedded: $("resizeEmbedded").checked,
        transcode: $("transcode").value,
        backup: $("backup").checked,
        dryRun: $("dryRun").checked,
    };
}

function appendLog(line) {
    const log = $("log");
    log.textContent = log.textContent ? log.textContent + "\n" + line : line;
    log.scrollTop = log.scrollHeight;
}

function resetOutput() {
    $("log").textContent = "";
    $("summary").textContent = "";
    $("error").textContent = "";
}

async function onChooseFolder() {
    const dir = await App().OpenFolder();
    if (dir) {
        $("dir").value = dir;
        $("pathLabel").textContent = dir;
        updateRunEnabled();
    }
}

async function onRun() {
    if ($("runBtn").dataset.running === "true") return;
    if (!$("dir").value) return;
    resetOutput();
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

function onDone(summary) {
    $("summary").textContent = summary || "";
    setRunning(false);
}

function onError(msg) {
    $("error").textContent = msg || "";
    setRunning(false);
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

    runtime().EventsOn("cf:progress", appendLog);
    runtime().EventsOn("cf:done", onDone);
    runtime().EventsOn("cf:error", onError);
    runtime().EventsOn("cf:state", (running) => setRunning(!!running));

    $("folderBtn").addEventListener("click", onChooseFolder);
    $("runBtn").addEventListener("click", onRun);
    $("cancelBtn").addEventListener("click", () => App().Cancel());
}

document.addEventListener("DOMContentLoaded", init);
