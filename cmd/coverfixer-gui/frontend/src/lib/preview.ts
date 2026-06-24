// Pure helpers for the library preview, ported from the old app.js/preview.js
// so they stay free of DOM and store coupling (and are unit-testable).

import type { Album, TagEdit } from './types';

/** A track shaped for display: per-track artist is shown only when it differs. */
export interface DisplayTrack {
  number: number;
  title: string;
  artist: string;
  duration: number;
  disc: number;
}

/** An album with the staged/global overlay applied, ready to render. */
export interface DisplayAlbum {
  title: string;
  artist: string;
  genre: string;
  year: string;
  artwork: string;
  staged: boolean;
  tracks: DisplayTrack[];
}

/** The sidebar metadata fields that overlay the preview when the group is on. */
export interface MetadataOverlay {
  enabled: boolean;
  albumArtist: string;
  genre: string;
}

/** "M:SS" (blank for non-positive durations), per the original preview.js. */
export function formatDuration(sec: number): string {
  if (!sec || sec <= 0) return '';
  sec = Math.round(sec);
  const m = Math.floor(sec / 60);
  const s = sec % 60;
  return m + ':' + String(s).padStart(2, '0');
}

/**
 * Group tracks by disc number, folding an unset disc (0) to 1, sorted by disc.
 * A single-disc album yields one group so the caller can skip "Disc N".
 */
export function discGroups<T extends { disc: number }>(
  tracks: T[],
): { disc: number; tracks: T[] }[] {
  const map = new Map<number, T[]>();
  for (const t of tracks) {
    const d = t.disc && t.disc > 0 ? t.disc : 1;
    if (!map.has(d)) map.set(d, []);
    map.get(d)!.push(t);
  }
  return [...map.entries()]
    .sort((a, b) => a[0] - b[0])
    .map(([disc, tr]) => ({ disc, tracks: tr }));
}

/**
 * Overlay pending metadata onto an album for display. Precedence (highest
 * wins): a staged per-album edit, then the sidebar's global Album Artist /
 * Genre (when the Metadata group is on), then the file tags. Per-track title /
 * number / artist are overlaid the same way, and a track's artist is shown only
 * when it differs from the album's effective artist.
 */
export function prepareAlbum(
  album: Album,
  edit: TagEdit | undefined,
  meta: MetadataOverlay,
): DisplayAlbum {
  const a: DisplayAlbum = {
    title: album.title,
    artist: album.artist,
    genre: album.genre,
    year: album.year,
    artwork: album.artwork,
    staged: false,
    tracks: [],
  };

  if (edit) {
    a.staged = true;
    a.title = edit.album;
    a.artist = edit.albumArtist;
    a.genre = edit.genre;
    a.year = edit.year;
  } else if (meta.enabled) {
    const artist = meta.albumArtist.trim();
    const genre = meta.genre.trim();
    if (artist) a.artist = artist;
    if (genre) a.genre = genre;
  }

  const albumArtist = a.artist;
  a.tracks = (album.tracks || []).map((t, j) => {
    const te = edit && edit.tracks[j];
    const artist = te ? te.artist : t.artist || '';
    const shown = artist && artist !== albumArtist ? artist : '';
    return {
      number: te ? te.trackNumber : t.number,
      title: te ? te.title : t.title,
      artist: shown,
      duration: t.duration,
      disc: t.disc,
    };
  });

  return a;
}

/** Build a TagEdit baseline from an album's on-disk tags (for edit-unchanged detection). */
export function editFromAlbum(album: Album): TagEdit {
  return {
    album: album.title || '',
    albumArtist: album.artist || '',
    genre: album.genre || '',
    year: album.year || '',
    tracks: (album.tracks || []).map((t) => ({
      path: t.path || '',
      title: t.title || '',
      artist: t.artist || '',
      trackNumber: t.number || 0,
    })),
  };
}

/** Structural equality of two staged TagEdits (unchanged => do not stage). */
export function editsEqual(a: TagEdit, b: TagEdit): boolean {
  if (
    a.album !== b.album ||
    a.albumArtist !== b.albumArtist ||
    a.genre !== b.genre ||
    a.year !== b.year
  ) {
    return false;
  }
  const at = a.tracks || [];
  const bt = b.tracks || [];
  if (at.length !== bt.length) return false;
  for (let i = 0; i < at.length; i++) {
    if (
      at[i].title !== bt[i].title ||
      at[i].artist !== bt[i].artist ||
      at[i].trackNumber !== bt[i].trackNumber
    ) {
      return false;
    }
  }
  return true;
}

/** Last path segment of dir (the folder's display name), tolerating / and \. */
export function folderName(dir: string): string {
  if (!dir) return '';
  const parts = dir.split(/[\\/]/).filter(Boolean);
  return parts[parts.length - 1] || dir;
}
