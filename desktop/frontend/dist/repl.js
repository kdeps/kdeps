"use strict";
// Model picker and composer autocomplete: the window's version of the CLI REPL's
// /model picker and tab completion. Relies on app.js globals (api, $, el, state).

const repl = { models: [], type: "all", descs: new Map(), cands: [], comp: null, idx: 0, seq: 0 };

async function refreshModelBtn() {
  try {
    const m = await api.CurrentModel();
    $("model-btn").textContent = m.model || "Choose model";
    $("model-backend").textContent = m.backend || "";
  } catch (e) { $("model-btn").textContent = "Choose model"; }
}

async function loadCommands() {
  try {
    repl.descs = new Map(((await api.Commands()) || []).map((c) => [c.name, c.desc || ""]));
  } catch (e) { repl.descs = new Map(); }
}

// ---- model picker ----

async function openModels() {
  $("models-confirm").hidden = true;
  try { repl.models = (await api.Models()) || []; } catch (e) { addMessage("error", String(e)); return; }
  $("models-q").value = "";
  renderModels();
  $("models").showModal();
  $("models-q").focus();
}

function renderTypes() {
  const counts = typeCounts(repl.models);
  const box = $("models-types");
  box.replaceChildren();
  for (const t of MODEL_TYPES) {
    const b = el("button", "", t + " " + (counts[t] || 0));
    b.type = "button";
    b.setAttribute("aria-pressed", String(repl.type === t));
    b.addEventListener("click", () => { repl.type = t; renderModels(); });
    box.appendChild(b);
  }
}

function renderModels() {
  renderTypes();
  const ul = $("models-list");
  ul.replaceChildren();
  for (const m of filterModels(repl.models, repl.type, $("models-q").value)) {
    const action = modelAction(m);
    const li = el("li", (m.current ? "current " : "") + (action === "needs-key" ? "off" : ""));
    li.tabIndex = 0;
    li.append(el("span", "mname", (m.favorite ? "* " : "") + m.name));
    const badges = el("span", "badges");
    badges.append(el("span", "badge", m.type));
    if (m.current) badges.append(el("span", "badge ok", "current"));
    else if (action === "download") badges.append(el("span", "badge", "not downloaded"));
    else if (action === "needs-key") badges.append(el("span", "badge", "no api key"));
    else badges.append(el("span", "badge ok", "ready"));
    li.append(badges);
    const pick = () => pickModel(m);
    li.addEventListener("click", pick);
    li.addEventListener("keydown", (ev) => { if (ev.key === "Enter") pick(); });
    ul.appendChild(li);
  }
}

function pickModel(m) {
  const action = modelAction(m);
  if (action === "use") return switchModel(m);
  const box = $("models-confirm");
  box.replaceChildren();
  const row = el("div", "row");
  const cancel = el("button", "", "Cancel");
  cancel.type = "button";
  cancel.addEventListener("click", () => { box.hidden = true; });
  if (action === "needs-key") {
    box.append(el("div", "", m.name + " runs on " + m.backend + ", which has no API key. Add one in Settings, API keys."));
    const open = el("button", "primary", "Open settings");
    open.type = "button";
    open.addEventListener("click", () => { $("models").close(); openSettings(); });
    row.append(cancel, open);
  } else {
    box.append(el("div", "", m.name + " is not on this machine. Download it now and switch? Progress shows in the chat."));
    const go = el("button", "primary", "Download and use");
    go.type = "button";
    go.addEventListener("click", () => switchModel(m));
    row.append(cancel, go);
  }
  box.append(row);
  box.hidden = false;
}

async function switchModel(m) {
  $("models").close();
  try {
    startCommandBlock("/model " + m.name);
    await api.SetModel(m.name);
    setRunning(true);
  } catch (e) { addMessage("error", String(e)); }
}

function startCommandBlock(echo) {
  addMessage("user", echo);
  state.turn = null;
  state.cmd = null;
}

function onModelEvent(e) {
  $("model-btn").textContent = e.model;
  $("model-backend").textContent = e.backend || "";
}

function onCommandOut(text) {
  if (!state.cmd) {
    state.cmd = el("pre", "cmdout");
    $("messages").appendChild(state.cmd);
  }
  state.cmd.textContent += text;
  scrollDown();
}

function onUIEvent(e) {
  if (e.text === "settings") openSettings();
  else if (e.text === "models") openModels();
  else if (e.text === "clear") { $("messages").replaceChildren(); }
  else if (e.text === "reload" && e.sessionId) openSession(e.sessionId);
}

function addSwitchModelButton(node) {
  const b = el("button", "", "Switch model");
  b.type = "button";
  b.addEventListener("click", openModels);
  node.appendChild(b);
}

// ---- autocomplete ----

function suggestOpen() { return !$("suggest").hidden; }

function closeSuggest() {
  $("suggest").hidden = true;
  repl.cands = [];
}

function renderSuggest() {
  const ul = $("suggest");
  ul.replaceChildren();
  repl.cands.forEach((c, i) => {
    const li = el("li");
    li.setAttribute("role", "option");
    li.setAttribute("aria-selected", String(i === repl.idx));
    li.append(el("span", "", c));
    const d = repl.descs.get(c);
    if (d) li.append(el("span", "desc", d));
    li.addEventListener("mousedown", (ev) => { ev.preventDefault(); repl.idx = i; acceptSuggest(); });
    ul.appendChild(li);
  });
  ul.hidden = repl.cands.length === 0;
  const sel = ul.querySelector('[aria-selected="true"]');
  if (sel) sel.scrollIntoView({ block: "nearest" });
}

async function updateSuggest() {
  const input = $("input");
  const pos = input.selectionStart;
  if (!shouldComplete(input.value, pos)) return closeSuggest();
  const seq = ++repl.seq;
  let res;
  try { res = await api.Complete(input.value, caretOffset(input.value, pos)); } catch (e) { return closeSuggest(); }
  if (seq !== repl.seq) return;
  repl.comp = res;
  repl.cands = ((res && res.candidates) || []).slice(0, 50);
  repl.idx = 0;
  renderSuggest();
}

function acceptSuggest() {
  const input = $("input");
  const choice = repl.cands[repl.idx];
  if (choice === undefined) return;
  const wasCommand = choice.startsWith("/") && !input.value.slice(0, input.selectionStart).includes(" ");
  const r = applyCompletion(input.value, input.selectionStart, repl.comp, wasCommand ? choice + " " : choice);
  input.value = r.text;
  input.setSelectionRange(r.pos, r.pos);
  closeSuggest();
  updateSuggest();
}

// Returns true when the key was consumed by the suggestion list.
function suggestHandleKey(ev) {
  if (!suggestOpen() || ev.isComposing) return false;
  const n = repl.cands.length;
  switch (ev.key) {
    case "ArrowDown": repl.idx = (repl.idx + 1) % n; renderSuggest(); break;
    case "ArrowUp": repl.idx = (repl.idx - 1 + n) % n; renderSuggest(); break;
    case "Tab": acceptSuggest(); break;
    case "Escape": closeSuggest(); break;
    case "Enter": {
      if (ev.shiftKey || $("input").value.trim() === repl.cands[repl.idx]) return false;
      acceptSuggest();
      break;
    }
    default: return false;
  }
  ev.preventDefault();
  return true;
}

$("model-btn").addEventListener("click", openModels);
$("models-close").addEventListener("click", () => $("models").close());
$("models-q").addEventListener("input", renderModels);
$("input").addEventListener("input", updateSuggest);
$("input").addEventListener("blur", () => setTimeout(closeSuggest, 100));
api.Ready().then(() => { refreshModelBtn(); loadCommands(); }).catch(() => {});
