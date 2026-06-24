// Single source of truth for the GUI, driven by Svelte 5 runes. Owns the view
// state machine, the run lifecycle wiring (cf:* events), the staged per-album
// edits, and the sidebar form. Components read derived getters; user actions
// are methods. Replaces the old imperative app.js.

import { app, runtime, waitForWails, Events } from './wails';
import {
  editsEqual,
  editFromAlbum,
  folderName,
  prepareAlbum,
  type DisplayAlbum,
} from './preview';
import type { Album, RunRequest, TagEdit } from './types';

export type Phase = 'empty' | 'loading' | 'ready';

/** Sidebar form fields, bound directly to inputs via bind:value/bind:checked. */
export interface Form {
  artSize: string; // "500" | "480" | "320" | "240"
  coverJpgSize: string;
  resizeEmbedded: boolean; // Resize Cover Art master toggle
  coverJpgGroup: boolean; // Extract cover.jpg master toggle
  transcodeGroup: boolean; // Transcode master toggle
  transcodeFormat: string; // "mp3" | "aac"
  transcodeQuality: string; // "320" | "256" | "192"
  metadataGroup: boolean; // Update Metadata master toggle
  albumArtist: string;
  genre: string;
  backup: boolean;
}

/** A track row being edited in the per-album modal. */
interface DraftTrack {
  title: string;
  artist: string;
  trackNumber: number;
}

/** The staged edit form for the modal. */
interface EditDraft {
  album: string;
  albumArtist: string;
  genre: string;
  year: string;
  tracks: DraftTrack[];
}

const ART_SIZES = ['500', '480', '320', '240'];

const EMPTY_FORM: Form = {
  artSize: '500',
  coverJpgSize: '500',
  resizeEmbedded: false,
  coverJpgGroup: true,
  transcodeGroup: false,
  transcodeFormat: 'mp3',
  transcodeQuality: '320',
  metadataGroup: false,
  albumArtist: '',
  genre: '',
  backup: false,
};

class Store {
  // --- view state machine -------------------------------------------------
  phase: Phase = $state('empty');
  running = $state(false);
  previewLoading = $state(false);

  // --- selected folder + library -----------------------------------------
  dir = $state('');
  albums = $state<Album[] | null>(null);

  // --- run output ---------------------------------------------------------
  summary = $state('');
  error = $state('');

  // --- sidebar form -------------------------------------------------------
  form = $state<Form>({ ...EMPTY_FORM });

  // --- staged per-album edits, keyed by index into `albums` ---------------
  stagedEdits = $state<Map<number, TagEdit>>(new Map());

  // --- per-album edit modal ----------------------------------------------
  editingIdx = $state<number | null>(null);
  editingDraft = $state<EditDraft | null>(null);
  /** Parallel to the draft's tracks; not reactive (read on Save only). */
  editingPaths: string[] = [];

  /** Bumped on every folder load to cancel stale in-flight fetches. */
  private loadToken = 0;

  // --- derived ------------------------------------------------------------

  /** The RunRequest assembled from the current form + staged edits. */
  get request(): RunRequest {
    const f = this.form;
    const coverJpg = f.coverJpgGroup;
    return {
      dir: this.dir,
      artSize: Number(f.artSize) || 0,
      coverJpgSize: Number(f.coverJpgSize) || 0,
      // jpegQuality is not exposed in the GUI; send 0 so the engine uses its
      // default (baseline 85), matching the original frontend's behaviour.
      jpegQuality: 0,
      recursive: true,
      renameStrayJpg: coverJpg,
      resizeCoverJpg: coverJpg,
      extractCover: coverJpg,
      resizeEmbedded: f.resizeEmbedded,
      transcode: f.transcodeGroup ? `${f.transcodeFormat}-${f.transcodeQuality}` : 'none',
      setGenre: f.metadataGroup,
      genre: f.genre,
      setAlbumArtist: f.metadataGroup,
      albumArtist: f.albumArtist,
      tagEdits: [...this.stagedEdits.values()],
      backup: f.backup,
      dryRun: false,
    };
  }

  /** Albums with staged/global metadata overlay applied, ready to render. */
  get visibleAlbums(): DisplayAlbum[] {
    if (!this.albums) return [];
    const meta = {
      enabled: this.form.metadataGroup,
      albumArtist: this.form.albumArtist,
      genre: this.form.genre,
    };
    return this.albums.map((a, i) => prepareAlbum(a, this.stagedEdits.get(i), meta));
  }

  /** "· N Albums, M Tracks" (singular-aware), or '' when nothing loaded. */
  get countsText(): string {
    if (!this.albums || this.albums.length === 0) return '';
    const aN = this.albums.length;
    let tN = 0;
    for (const a of this.albums) tN += (a.tracks || []).length;
    return (
      '· ' +
      aN +
      ' Album' +
      (aN === 1 ? '' : 's') +
      ', ' +
      tN +
      ' Track' +
      (tN === 1 ? '' : 's')
    );
  }

  get folderDisplayName(): string {
    return folderName(this.dir);
  }

  get canRun(): boolean {
    return this.phase === 'ready' && !this.running && !!this.dir;
  }

  get editOpen(): boolean {
    return this.editingIdx !== null;
  }

  // --- lifecycle ----------------------------------------------------------

  /** Wire up Wails globals + events and seed the form. Called once on mount. */
  async init(): Promise<void> {
    const ready = await waitForWails();
    if (!ready) {
      this.error = 'Failed to connect to the backend runtime.';
      return;
    }
    try {
      this.seedDefaults(await app()!.DefaultRequest());
    } catch (e) {
      // Non-fatal: leave the form at its built-in defaults.
      console.error(e);
    }

    const rt = runtime()!;
    rt.EventsOn(Events.Done, (s) => this.onDone(s as string));
    rt.EventsOn(Events.Error, (m) => this.onError(m as string));
    rt.EventsOn(Events.State, (r) => this.onState(r as boolean));
    // A folder dropped on the window or Dock icon while running. A
    // launch-by-drop is pulled separately via InitialFolder below.
    rt.EventsOn(Events.Folder, (dir) => {
      if (dir) this.applyFolder(dir as string);
    });

    try {
      const initial = await app()!.InitialFolder();
      if (initial) await this.applyFolder(initial);
      // else: stays 'empty' — the empty state is shown.
    } catch (e) {
      console.error(e);
    }
  }

  /** Seed the sidebar form from the Go-side defaults (single source of truth). */
  seedDefaults(req: RunRequest): void {
    this.form.artSize = ART_SIZES.includes(String(req.artSize)) ? String(req.artSize) : '500';
    this.form.coverJpgSize = ART_SIZES.includes(String(req.coverJpgSize))
      ? String(req.coverJpgSize)
      : '500';
    this.form.backup = !!req.backup;
    // Transcode: none vs. the selected Format × Quality combination.
    const tc = req.transcode && req.transcode !== 'none' ? req.transcode : 'mp3-320';
    const [fmt, qual] = tc.split('-');
    this.form.transcodeFormat = fmt || 'mp3';
    this.form.transcodeQuality = qual || '320';
    this.form.transcodeGroup = !!(req.transcode && req.transcode !== 'none');
    this.form.metadataGroup = !!req.setGenre;
    this.form.resizeEmbedded = !!req.resizeEmbedded;
    this.form.coverJpgGroup = !!(req.renameStrayJpg || req.resizeCoverJpg || req.extractCover);
  }

  // --- folder selection ---------------------------------------------------

  /** Open the native picker and load the chosen folder. */
  async chooseFolder(): Promise<void> {
    const dir = await app()!.OpenFolder();
    if (dir) await this.applyFolder(dir);
  }

  /**
   * Select a folder (shared by picker, window drop, launch-by-drop): set the
   * path, prefill metadata from the first audio file, then load the preview.
   * A no-op while a run is in flight (folder changes mid-run would race the
   * preview and the engine). Cancels any prior in-flight load via loadToken.
   */
  async applyFolder(dir: string): Promise<void> {
    if (!dir || this.running) return;
    const token = ++this.loadToken;
    this.dir = dir;
    this.stagedEdits.clear();
    this.stagedEdits = new Map(this.stagedEdits); // notify: Map clear is structural
    this.error = '';
    this.summary = '';
    this.albums = null; // first-load placeholder for "Reading library…"
    this.previewLoading = true;
    this.phase = 'loading';

    // Prefill the metadata fields from the first audio file's existing tags.
    try {
      const m = await app()!.ReadFirstMetadata(dir);
      if (token !== this.loadToken) return;
      this.form.albumArtist = m.albumArtist || '';
      this.form.genre = m.genre || '';
    } catch (e) {
      console.error(e);
    }
    if (token !== this.loadToken) return;

    try {
      const albums = await app()!.ReadLibrary(dir, true);
      if (token !== this.loadToken) return;
      this.albums = albums;
    } catch (e) {
      if (token !== this.loadToken) return;
      this.albums = [];
      this.error = 'Could not read library: ' + String(e);
    } finally {
      if (token === this.loadToken) {
        this.previewLoading = false;
        this.phase = 'ready';
      }
    }
  }

  /**
   * Re-read the library after a run (runs always mutate). Keeps the current
   * albums visible until the fresh ones arrive (no empty-state flash). On
   * success staged edits were written, so they are cleared by the caller.
   */
  private async rescan(): Promise<void> {
    if (!this.dir) return;
    const token = ++this.loadToken;
    try {
      const albums = await app()!.ReadLibrary(this.dir, true);
      if (token !== this.loadToken) return;
      this.albums = albums;
    } catch {
      // keep previous albums
    }
  }

  // --- run lifecycle ------------------------------------------------------

  async run(): Promise<void> {
    if (!this.canRun) return;
    this.error = '';
    this.summary = '';
    this.running = true; // optimistic; cf:state reconciles on completion
    try {
      // Run resolves once the engine has started; completion arrives via
      // cf:done/cf:error. A rejection means the run never started.
      await app()!.Run(this.request);
    } catch (e) {
      this.error = String(e);
      this.running = false;
    }
  }

  cancel(): void {
    app()!.Cancel();
  }

  // --- cf:* event handlers ------------------------------------------------

  private onState(running: boolean): void {
    this.running = running;
  }

  private onDone(summary: string): void {
    this.summary = summary || '';
    this.running = false;
    // Staged edits were written by a successful Run; clear so the rescan
    // (which picks them up from disk) shows no badge.
    this.stagedEdits.clear();
    this.stagedEdits = new Map(this.stagedEdits);
    void this.rescan();
  }

  private onError(msg: string): void {
    this.error = msg || '';
    this.running = false;
    // Keep staged edits on failure so the user can retry without re-entering
    // them; the rescan reflects whatever the engine actually wrote.
    void this.rescan();
  }

  // --- per-album edit modal ----------------------------------------------

  openEdit(idx: number): void {
    if (!this.albums || idx < 0 || idx >= this.albums.length) return;
    const album = this.albums[idx];
    const edit = this.stagedEdits.get(idx);
    const tracks = album.tracks || [];
    this.editingIdx = idx;
    this.editingPaths = tracks.map((t) => t.path || '');
    this.editingDraft = edit
      ? {
          album: edit.album,
          albumArtist: edit.albumArtist,
          genre: edit.genre,
          year: edit.year,
          tracks: tracks.map((t, j) => {
            const te = edit.tracks[j];
            return {
              title: te ? te.title : t.title || '',
              artist: te ? te.artist : t.artist || '',
              trackNumber: te ? te.trackNumber : t.number || 0,
            };
          }),
        }
      : {
          album: album.title || '',
          albumArtist: album.artist || '',
          genre: album.genre || '',
          year: album.year || '',
          tracks: tracks.map((t) => ({
            title: t.title || '',
            artist: t.artist || '',
            trackNumber: t.number || 0,
          })),
        };
  }

  /** Stage the modal's draft for the next Run (if it differs from baseline). */
  stageEdit(): void {
    if (this.editingIdx === null || !this.albums || !this.editingDraft) return;
    const idx = this.editingIdx;
    const draft = this.editingDraft;
    const desired: TagEdit = {
      album: draft.album,
      albumArtist: draft.albumArtist,
      genre: draft.genre,
      year: draft.year,
      tracks: draft.tracks.map((t, j) => ({
        path: this.editingPaths[j] || '',
        title: t.title,
        artist: t.artist,
        trackNumber: t.trackNumber,
      })),
    };
    const baseline = editFromAlbum(this.albums[idx]);
    if (editsEqual(desired, baseline)) {
      this.stagedEdits.delete(idx);
    } else {
      this.stagedEdits.set(idx, desired);
    }
    this.stagedEdits = new Map(this.stagedEdits); // notify
    this.closeEdit();
  }

  /** Unstage any pending edit for the album being edited, then close. */
  revertEdit(): void {
    if (this.editingIdx !== null) {
      this.stagedEdits.delete(this.editingIdx);
      this.stagedEdits = new Map(this.stagedEdits); // notify
    }
    this.closeEdit();
  }

  closeEdit(): void {
    this.editingIdx = null;
    this.editingDraft = null;
    this.editingPaths = [];
  }
}

export const store = new Store();
