"use strict";

// Pure helpers for the composer (autocomplete) and the model picker.

const MODEL_TYPES = ["all", "llamafile", "gguf", "ollama", "cloud"];

// Offsets sent to Go are code points, not UTF-16 units.
function caretOffset(text, utf16Pos) {
  return Array.from(text.slice(0, utf16Pos)).length;
}

// Complete only slash commands and @file tokens; plain prose is left alone.
function shouldComplete(text, utf16Pos) {
  if (text.startsWith("/")) return true;
  const token = text.slice(0, utf16Pos).split(/\s/).pop();
  return token.startsWith("@");
}

// Replace the last comp.replace code points before the caret with choice.
function applyCompletion(text, utf16Pos, comp, choice) {
  const head = Array.from(text.slice(0, utf16Pos));
  const keep = head.slice(0, Math.max(0, head.length - comp.replace)).join("");
  const next = keep + choice;
  return { text: next + text.slice(utf16Pos), pos: next.length };
}

function filterModels(models, type, query) {
  const q = (query || "").trim().toLowerCase();
  return models.filter((m) => {
    if (type && type !== "all" && m.type !== type) return false;
    if (!q) return true;
    return m.name.toLowerCase().includes(q) || (m.repo || "").toLowerCase().includes(q);
  });
}

function typeCounts(models) {
  const out = { all: models.length };
  for (const t of MODEL_TYPES.slice(1)) out[t] = models.filter((m) => m.type === t).length;
  return out;
}

// What picking a model does: switch now, download first, or needs an API key.
function modelAction(m) {
  if (m.type === "cloud") return m.enabled ? "use" : "needs-key";
  return m.downloaded ? "use" : "download";
}

if (typeof module !== "undefined") {
  module.exports = { MODEL_TYPES, caretOffset, shouldComplete, applyCompletion, filterModels, typeCounts, modelAction };
}
