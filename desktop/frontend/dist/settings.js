"use strict";
// Settings modal: Appearance, All settings, Prompts and harness, Instructions,
// Memories, Profile. Relies on app.js globals (api, $, el).

const TABS = [
  ["appearance", "Appearance", renderAppearance],
  ["all", "All settings", renderAllSettings],
  ["harness", "Prompts and harness", renderHarness],
  ["instructions", "Custom instructions", renderInstructions],
  ["memories", "Memories", renderMemories],
  ["profile", "Work profile", renderProfile],
];

function status(parent) {
  const s = el("div", "status");
  parent.appendChild(s);
  return (msg, isErr) => { s.textContent = msg; s.className = "status" + (isErr ? " err" : ""); };
}

function selectTab(id) {
  for (const b of document.querySelectorAll("#settings-tabs button")) {
    b.setAttribute("aria-selected", String(b.dataset.tab === id));
  }
  const pane = $("settings-pane");
  pane.replaceChildren();
  const tab = TABS.find((t) => t[0] === id);
  Promise.resolve(tab[2](pane)).catch((e) => pane.appendChild(el("div", "status err", String(e))));
}

function openSettings() {
  const nav = $("settings-tabs");
  if (!nav.children.length) {
    for (const [id, label] of TABS) {
      const b = el("button", "", label);
      b.dataset.tab = id;
      b.setAttribute("role", "tab");
      b.addEventListener("click", () => selectTab(id));
      nav.appendChild(b);
    }
  }
  $("settings").showModal();
  selectTab("appearance");
}

function lsGet(k, d) { try { return localStorage.getItem(k) || d; } catch (_) { return d; } }
function lsSet(k, v) { try { localStorage.setItem(k, v); } catch (_) { /* storage unavailable */ } }

function applyAppearance() {
  const root = document.documentElement;
  const theme = lsGet("kdeps.theme", "system");
  if (theme === "system") root.removeAttribute("data-theme"); else root.dataset.theme = theme;
  const accent = lsGet("kdeps.accent", "cyan");
  if (accent === "cyan") root.removeAttribute("data-accent"); else root.dataset.accent = accent;
  root.dataset.density = lsGet("kdeps.density", "comfortable");
}

function choice(parent, label, key, options) {
  const row = el("div", "field");
  const lab = el("label", "", label);
  const sel = el("select");
  for (const [v, t] of options) {
    const o = el("option", "", t);
    o.value = v;
    sel.appendChild(o);
  }
  sel.value = lsGet(key, options[0][0]);
  lab.htmlFor = sel.id = "ap-" + key.replace(/\W/g, "");
  sel.addEventListener("change", () => { lsSet(key, sel.value); applyAppearance(); });
  row.append(lab, sel);
  parent.appendChild(row);
}

function renderAppearance(pane) {
  choice(pane, "Theme", "kdeps.theme", [["system", "Match system"], ["dark", "Dark"], ["light", "Light"]]);
  choice(pane, "Accent", "kdeps.accent", [["cyan", "Cyan"], ["pink", "Pink"], ["yellow", "Yellow"]]);
  choice(pane, "Density", "kdeps.density", [["comfortable", "Comfortable"], ["compact", "Compact"]]);
  pane.appendChild(el("div", "hint", "Appearance is stored on this device only."));
}

async function renderAllSettings(pane) {
  const fields = (await api.Settings()) || [];
  const say = status(pane);
  let group = null;
  for (const f of fields) {
    if (f.group !== group) {
      group = f.group;
      pane.appendChild(el("div", "eyebrow", group || "general"));
    }
    pane.appendChild(settingRow(f, say));
  }
  pane.appendChild(el("div", "hint", "Changes are written to config.yaml. Empty a field to remove it. Backend and model changes apply on the next new chat."));
}

function settingRow(f, say) {
  const row = el("div", "field");
  const lab = el("label", "", f.label);
  lab.title = f.path;
  let input;
  if (f.type === "bool") {
    input = el("input");
    input.type = "checkbox";
    input.checked = f.value === true;
  } else {
    input = el("input");
    input.type = f.secret ? "password" : f.type === "string" ? "text" : "number";
    if (f.type === "number") input.step = "any";
    if (f.secret) input.placeholder = f.set ? "stored (type to replace)" : "not set";
    else if (f.value !== null && f.value !== undefined) input.value = String(f.value);
  }
  lab.htmlFor = input.id = "set-" + f.path;
  input.addEventListener("change", async () => {
    const value = f.type === "bool" ? input.checked : input.value;
    try {
      await api.SetSetting(f.path, value);
      say("Saved " + f.path);
    } catch (e) { say(String(e), true); }
  });
  row.append(lab, input);
  return row;
}

async function renderHarness(pane) {
  const say = status(pane);
  const presets = (await api.Presets()) || [];
  if (presets.length) {
    pane.appendChild(el("div", "eyebrow", "Presets"));
    for (const p of presets) {
      const card = el("div", "card");
      const top = el("div", "top");
      const b = el("button", "", "Apply");
      b.addEventListener("click", async () => {
        try { await api.ApplyPreset(p.Name); say("Applied " + p.Name); selectTab("harness"); } catch (e) { say(String(e), true); }
      });
      top.append(el("strong", "", p.Name), b);
      card.append(top, el("div", "hint", p.Description || ""));
      pane.appendChild(card);
    }
  }
  pane.appendChild(el("div", "eyebrow", "Harness sections"));
  for (const s of (await api.HarnessSections()) || []) {
    const card = el("div", "card");
    const top = el("div", "top");
    const en = el("input"); en.type = "checkbox"; en.checked = s.enabled; en.id = "hs-en-" + s.name;
    const rm = el("input"); rm.type = "checkbox"; rm.checked = s.remind; rm.id = "hs-rm-" + s.name;
    const l1 = el("label", "", "enabled"); l1.htmlFor = en.id;
    const l2 = el("label", "", "remind each turn"); l2.htmlFor = rm.id;
    const save = async () => {
      try { await api.SetHarnessSection(s.name, en.checked, rm.checked); say("Saved " + s.name); } catch (e) { say(String(e), true); }
    };
    en.addEventListener("change", save);
    rm.addEventListener("change", save);
    top.append(el("strong", "", s.name), el("span", "hint", s.kind), en, l1, rm, l2);
    const body = el("details");
    body.append(el("summary", "hint", "text"), el("pre", "hint", s.body));
    card.append(top, body);
    pane.appendChild(card);
  }
}

async function renderInstructions(pane) {
  const say = status(pane);
  const ta = el("textarea");
  ta.value = (await api.Instructions()) || "";
  ta.setAttribute("aria-label", "Custom instructions");
  const save = el("button", "primary", "Save");
  save.addEventListener("click", async () => {
    try { await api.SetInstructions(ta.value); say("Saved to KDEPS.md in the workspace"); } catch (e) { say(String(e), true); }
  });
  pane.prepend(el("div", "hint", "Instructions for this workspace, saved as KDEPS.md and sent with every chat."));
  pane.insertBefore(ta, pane.lastChild);
  pane.insertBefore(save, pane.lastChild);
}

async function renderMemories(pane) {
  const say = status(pane);
  const add = el("div", "field");
  const key = el("input"); key.type = "text"; key.placeholder = "key"; key.setAttribute("aria-label", "Memory key");
  const val = el("input"); val.type = "text"; val.placeholder = "value"; val.setAttribute("aria-label", "Memory value");
  const btn = el("button", "primary", "Save memory");
  btn.addEventListener("click", async () => {
    if (!key.value.trim()) return;
    try { await api.SaveMemory(key.value.trim(), val.value); selectTab("memories"); } catch (e) { say(String(e), true); }
  });
  add.append(key, val);
  pane.append(add, btn);
  for (const m of (await api.ListMemory()) || []) {
    const card = el("div", "card");
    const top = el("div", "top");
    const del = el("button", "", "Delete");
    del.addEventListener("click", async () => {
      try { await api.DeleteMemory(m.key); selectTab("memories"); } catch (e) { say(String(e), true); }
    });
    top.append(el("strong", "", m.key), del);
    card.append(top, el("div", "hint", m.value));
    pane.appendChild(card);
  }
}

function renderProfile(pane) {
  const say = status(pane);
  pane.appendChild(el("div", "hint", "A work profile bundles tuning, harness settings and skills into one file you can move between machines."));
  const exp = el("button", "primary", "Export profile");
  exp.addEventListener("click", async () => {
    try { const p = await api.SaveProfile(); if (p) say("Exported to " + p); } catch (e) { say(String(e), true); }
  });
  const imp = el("button", "", "Import profile");
  imp.addEventListener("click", async () => {
    try { const p = await api.LoadProfile(); if (p) say("Imported " + p); } catch (e) { say(String(e), true); }
  });
  pane.append(exp, imp);
}

$("open-settings").addEventListener("click", openSettings);
$("settings-close").addEventListener("click", () => $("settings").close());
applyAppearance();
