"use strict";
// Pure helpers for the chat list (no DOM), shared with node tests.

// folderGroups groups all-folder search hits by their folder, keeping the
// newest-first order of the hits: the folder with the most recent chat comes
// first, the current workspace is flagged.
function folderGroups(hits, workspace) {
  const groups = [];
  const byDir = new Map();
  for (const h of hits || []) {
    const dir = h.session.cwd || "";
    let g = byDir.get(dir);
    if (!g) {
      g = { dir, name: folderName(dir), current: dir === workspace, hits: [] };
      byDir.set(dir, g);
      groups.push(g);
    }
    g.hits.push(h);
  }
  return groups;
}

// folderName is the last path segment of dir, for a group header.
function folderName(dir) {
  const parts = (dir || "").split(/[\\/]/).filter(Boolean);
  return parts.length ? parts[parts.length - 1] : dir || "?";
}

if (typeof module !== "undefined") module.exports = { folderGroups, folderName };
