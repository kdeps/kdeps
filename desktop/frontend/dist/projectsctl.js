"use strict";
// Pure helpers for the Projects view (no DOM), shared with node tests.

// groupProjects splits the list into the workspace's own projects and
// installed registry packages, each sorted by name.
function groupProjects(list) {
  const byName = (a, b) => a.name.localeCompare(b.name);
  const all = list || [];
  return {
    workspace: all.filter((p) => !p.installed).sort(byName),
    installed: all.filter((p) => p.installed).sort(byName),
  };
}

// componentBody converts the input form's text values to the declared types.
function componentBody(inputs, values) {
  const out = {};
  for (const inp of inputs || []) {
    const raw = values[inp.name];
    if (raw === undefined || raw === "") {
      if (inp.required) throw new Error(inp.name + " is required");
      continue;
    }
    if (inp.type === "integer" || inp.type === "number") {
      const n = Number(raw);
      if (Number.isNaN(n) || (inp.type === "integer" && !Number.isInteger(n))) throw new Error(inp.name + " must be a " + (inp.type === "integer" ? "whole number" : "number"));
      out[inp.name] = n;
    } else if (inp.type === "boolean") {
      out[inp.name] = raw === true || raw === "true";
    } else {
      out[inp.name] = raw;
    }
  }
  return out;
}

// shownFields lists the fields the editor renders for an action: the ones
// with a value, in the type's order. addableFields is the rest.
function shownFields(kind, action, extra) {
  const set = new Set(Object.keys(action || {}).concat(extra || []));
  return (kind ? kind.fields : []).filter((f) => set.has(f.name));
}

function addableFields(kind, action, extra) {
  const set = new Set(Object.keys(action || {}).concat(extra || []));
  return (kind ? kind.fields : []).filter((f) => !set.has(f.name));
}

// controlFor picks the input control for a field type.
function controlFor(field) {
  switch (field.type) {
    case "boolean": return "checkbox";
    case "enum": return "select";
    case "text": case "yaml": return "textarea";
    case "integer": case "number": return "number";
    default: return "text";
  }
}

// formFieldValue turns a control's value into the form value sent to Go.
// Numbers stay text; the backend converts and reports bad input by name.
function formFieldValue(field, value) {
  if (field.type === "boolean") return value === true;
  return value == null ? "" : String(value);
}

// curlFor builds a copyable request for a served route. The token stays an
// env reference so it never appears on screen.
function curlFor(url, route) {
  const method = (route.methods && route.methods[0]) || "GET";
  const parts = ["curl"];
  if (method !== "GET") parts.push("-X " + method);
  parts.push('-H "Authorization: Bearer $KDEPS_API_AUTH_TOKEN"');
  if (method === "POST" || method === "PUT" || method === "PATCH") {
    parts.push("-H 'Content-Type: application/json'", "-d '{}'");
  }
  parts.push(url.replace(/\/$/, "") + route.path);
  return parts.join(" ");
}

// progressLabel describes one run_progress event.
function progressLabel(ev) {
  const what = { "resource.started": "running", "resource.completed": "done", "resource.skipped": "skipped",
    "resource.failed": "failed", "resource.retrying": "retrying", "workflow.started": "started",
    "workflow.completed": "finished", "workflow.failed": "failed" }[ev.status] || ev.status;
  const who = ev.summary ? ev.summary + ": " : "";
  return who + what + (ev.text ? " - " + ev.text : "");
}

// resourceTargets lists the actionIds a manifest's target can point at.
function resourceTargets(resources) {
  return (resources || []).map((r) => r.actionId).filter(Boolean);
}

// trimLog keeps the newest max lines.
function trimLog(lines, max) {
  return lines.length > max ? lines.slice(lines.length - max) : lines;
}

if (typeof module !== "undefined") {
  module.exports = { groupProjects, componentBody, shownFields, addableFields,
    controlFor, formFieldValue, curlFor, progressLabel, resourceTargets, trimLog };
}
