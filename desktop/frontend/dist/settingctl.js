"use strict";
// Picks the control for a settings field from its schema metadata.

function settingControl(f) {
  if (f.type === "bool") return "checkbox";
  if (f.secret) return "password";
  if (f.options && f.options.length) return "select";
  if ((f.type === "int" || f.type === "number") && f.min != null && f.max != null) return "range";
  if ((f.suggestions && f.suggestions.length) || f.path === "defaults.timezone") return "suggest";
  return f.type === "string" ? "text" : "number";
}

// selectChoices keeps a stored value that is not in the list selectable, so
// opening the form never silently rewrites an unusual config.
function selectChoices(f) {
  const out = [""].concat(f.options);
  const cur = f.value == null ? "" : String(f.value);
  if (cur && !out.includes(cur)) out.push(cur);
  return out;
}

if (typeof module !== "undefined") module.exports = { settingControl, selectChoices };
