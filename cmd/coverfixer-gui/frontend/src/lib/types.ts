// Wire types mirroring the Go structs in internal/core and internal/ui.
// These are JSON-friendly shapes sent over the Wails binding boundary; keep
// them in sync with the generated models in frontend/wailsjs/go/models.ts.

/** core.Track — one audio file's display metadata. */
export interface Track {
  number: number;
  title: string;
  artist: string;
  duration: number; // seconds
  disc: number; // 0 when unset
  path: string;
}

/** core.Album — a display-only grouping for the library preview. */
export interface Album {
  title: string;
  artist: string;
  genre: string;
  year: string;
  artwork: string; // base64 data: URL, empty when none
  tracks: Track[];
}

/** core.FirstMetadata — prefilled tags from the first audio file. */
export interface FirstMetadata {
  genre: string;
  albumArtist: string;
}

/** core.TrackTagEdit — per-file portion of a staged TagEdit. */
export interface TrackTagEdit {
  path: string;
  title: string;
  artist: string;
  trackNumber: number;
}

/** core.TagEdit — staged per-album metadata, applied on Run. */
export interface TagEdit {
  album: string;
  albumArtist: string;
  genre: string;
  year: string;
  tracks: TrackTagEdit[];
}

/** ui.RunRequest — the wire format the frontend sends to start a run. */
export interface RunRequest {
  dir: string;
  artSize: number;
  coverJpgSize: number;
  jpegQuality: number;
  recursive: boolean;
  renameStrayJpg: boolean;
  resizeCoverJpg: boolean;
  extractCover: boolean;
  resizeEmbedded: boolean;
  transcode: string; // "none" | "<mp3|aac>-<320|256|192>"
  setGenre: boolean;
  genre: string;
  setAlbumArtist: boolean;
  albumArtist: string;
  tagEdits: TagEdit[];
  backup: boolean;
  dryRun: boolean;
}
