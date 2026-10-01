"use strict";
// Wails injects window.go (bound Go methods) and window.runtime (events).
const api = window.go.main.App;
const rt = window.runtime;
const $ = (id) => document.getElementById(id);

const state = { files: [], current: null, pending: null, turn: null, tools: new Map() };

function el(tag, cls, text) {
  const e = document.createElement(tag);
  if (cls) e.className = cls;
  if (text !== undefined) e.textContent = text;
  return e;
}

function scrollDown() {
  const m = $("messages");
  m.scrollTop = m.scrollHeight;
}

function addMessage(cls, text) {
  const e = el("div", "msg " + cls, text);
  $("messages").appendChild(e);
  scrollDown();
  return e;
}

function setRunning(on) {
  $("send").hidden = on;
  $("stop").hidden = !on;
}

async function refreshSessions() {
  const q = $("search").value;
  let hits = [];
  try { hits = (await api.SearchSessions(q)) || []; } catch (e) { addMessage("error", String(e)); }
  const ul = $("sessions");
  ul.replaceChildren();
  for (const h of hits) {
    const li = el("li");
    li.tabIndex = 0;
    li.dataset.id = h.session.id;
    li.append(el("div", "title", h.session.name || h.session.firstPrompt || h.session.id),
              el("div", "snippet", h.snippet || ""));
    if (h.session.id === state.current) li.classList.add("active");
    const open = () => openSession(h.session.id);
    li.addEventListener("click", open);
    li.addEventListener("keydown", (ev) => { if (ev.key === "Enter") open(); });
    ul.appendChild(li);
  }
}

async function openSession(id) {
  try {
    const msgs = (await api.LoadSession(id)) || [];
    $("messages").replaceChildren();
    for (const m of msgs) addMessage(m.role === "user" ? "user" : "assistant", m.content);
    state.current = id;
    refreshSessions();
  } catch (e) { addMessage("error", String(e)); }
}

async function newChat() {
  try { await api.NewChat(); } catch (e) { addMessage("error", String(e)); return; }
  $("messages").replaceChildren();
  state.current = null;
  state.turn = null;
  refreshSessions();
}

function renderAttachments() {
  const box = $("attachments");
  box.replaceChildren();
  for (const f of state.files) {
    const c = el("span", "chip", f.split(/[\\/]/).pop() + " x");
    c.title = f;
    c.addEventListener("click", () => { state.files = state.files.filter((x) => x !== f); renderAttachments(); });
    box.appendChild(c);
  }
}

function addFiles(paths) {
  for (const p of paths || []) if (!state.files.includes(p)) state.files.push(p);
  renderAttachments();
}

async function send(ev) {
  ev.preventDefault();
  const text = $("input").value.trim();
  if (!text) return;
  addMessage("user", text);
  const files = state.files;
  $("input").value = "";
  state.files = [];
  renderAttachments();
  state.turn = null;
  try {
    await api.Send(text, files);
    setRunning(true);
  } catch (e) { addMessage("error", String(e)); }
}

function onEvent(e) {
  switch (e.kind) {
    case "token":
      if (!state.turn) state.turn = addMessage("assistant", "");
      state.turn.textContent += e.text;
      scrollDown();
      break;
    case "narration":
      addMessage("narration", e.text);
      state.turn = null;
      break;
    case "tool_start": {
      const d = el("details", "tool");
      d.append(el("summary", "", e.summary || e.tool), el("pre", "", e.args || ""));
      $("messages").appendChild(d);
      state.tools.set(e.callId, d);
      state.turn = null;
      scrollDown();
      break;
    }
    case "tool_end": {
      const d = state.tools.get(e.callId);
      if (d) {
        if (e.error) d.classList.add("err");
        d.append(el("pre", "", e.result || ""));
      }
      break;
    }
    case "approval":
      askApproval(e.approval);
      break;
    case "error":
      addMessage("error", e.text);
      setRunning(false);
      break;
    case "turn_end":
      state.current = e.sessionId || state.current;
      state.turn = null;
      setRunning(false);
      refreshSessions();
      break;
  }
}

function askApproval(a) {
  $("approval-title").textContent = a.kind === "path" ? "Allow access outside the workspace?" : "Allow tool: " + a.tool;
  $("approval-body").textContent = a.kind === "path" ? a.path + "\nworkspace: " + a.root : (a.summary || "") + "\n" + (a.args || "");
  state.pending = a.id;
  $("approval").showModal();
}

function answer(choice) {
  const id = state.pending;
  $("approval").close();
  if (id) api.Approve(id, choice).catch((e) => addMessage("error", String(e)));
}

async function refreshWorkspace() {
  const ws = await api.Workspace();
  $("workspace").textContent = ws;
  $("workspace").title = ws;
}

async function switchWorkspace() {
  try {
    const dir = await api.PickWorkspace();
    if (dir) { state.current = null; $("messages").replaceChildren(); await refreshWorkspace(); refreshSessions(); }
  } catch (e) { addMessage("error", String(e)); }
}

let dragDepth = 0;
window.addEventListener("dragenter", () => { dragDepth++; $("drop").hidden = false; });
window.addEventListener("dragleave", () => { if (--dragDepth <= 0) { dragDepth = 0; $("drop").hidden = true; } });
window.addEventListener("drop", () => { dragDepth = 0; $("drop").hidden = true; });
window.addEventListener("dragover", (e) => e.preventDefault());

$("composer").addEventListener("submit", send);
$("input").addEventListener("keydown", (ev) => {
  if (ev.key === "Enter" && !ev.shiftKey && !ev.isComposing) { ev.preventDefault(); $("composer").requestSubmit(); }
});
$("stop").addEventListener("click", () => api.Cancel());
$("new-chat").addEventListener("click", newChat);
$("workspace").addEventListener("click", switchWorkspace);
$("attach").addEventListener("click", async () => addFiles(await api.PickFiles()));
$("search").addEventListener("input", refreshSessions);
$("ap-once").addEventListener("click", () => answer("once"));
$("ap-always").addEventListener("click", () => answer("always"));
$("ap-deny").addEventListener("click", () => answer("deny"));
$("approval").addEventListener("cancel", (ev) => { ev.preventDefault(); answer("deny"); });

rt.EventsOn("kdeps:event", onEvent);
rt.EventsOn("kdeps:files", addFiles);
api.Ready().then(() => { refreshWorkspace(); refreshSessions(); })
  .catch((e) => addMessage("error", "startup failed: " + e));
