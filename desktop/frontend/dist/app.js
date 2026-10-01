"use strict";
// Wails injects window.go (bound Go methods) and window.runtime (events).
const api = window.go.main.App;
const rt = window.runtime;
const $ = (id) => document.getElementById(id);

const state = { files: [], current: null, pending: null, turn: null, tools: new Map(), cmd: null, draft: "" };

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

// Assistant text is Markdown; everything else is plain text.
function addMessage(cls, text) {
  const e = el("div", "msg " + cls);
  if (cls === "assistant") {
    e.raw = text;
    renderMarkdown(e, text);
  } else {
    e.textContent = text;
  }
  $("messages").appendChild(e);
  scrollDown();
  return e;
}

function setRunning(on) {
  $("send").hidden = on;
  $("stop").hidden = !on;
}

// Streamed tokens re-render at most once per frame.
function appendAssistant(node, text) {
  node.raw += text;
  if (node.pending) return;
  node.pending = true;
  requestAnimationFrame(() => {
    node.pending = false;
    renderMarkdown(node, node.raw);
    scrollDown();
  });
}

async function refreshSessions() {
  const q = $("search").value;
  let hits = [];
  try { hits = (await api.SearchSessions(q)) || []; } catch (e) { addMessage("error", String(e)); }
  const ul = $("sessions");
  ul.replaceChildren();
  if (!state.current && state.draft) {
    const d = el("li", "draft active");
    d.append(el("div", "title", state.draft), el("div", "snippet", "Current chat (not saved yet)"));
    ul.appendChild(d);
  }
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
    const del = el("button", "del", "Delete");
    del.type = "button";
    del.setAttribute("aria-label", "Delete chat");
    del.addEventListener("click", (ev) => {
      ev.stopPropagation();
      if (del.dataset.armed !== "1") {
        del.dataset.armed = "1";
        del.textContent = "Sure?";
        setTimeout(() => { del.dataset.armed = ""; del.textContent = "Delete"; }, 3000);
        return;
      }
      removeSession(h.session.id);
    });
    li.appendChild(del);
    ul.appendChild(li);
  }
}

async function removeSession(id) {
  try {
    await api.DeleteSession(id);
    if (id === state.current) {
      $("messages").replaceChildren();
      state.current = null;
    }
  } catch (e) { addMessage("error", String(e)); }
  refreshSessions();
}

async function openSession(id) {
  try {
    const msgs = (await api.LoadSession(id)) || [];
    $("messages").replaceChildren();
    state.draft = "";
    for (const m of msgs) addMessage(m.role === "user" ? "user" : "assistant", m.content);
    state.current = id;
    refreshSessions();
  } catch (e) { addMessage("error", String(e)); }
}

async function newChat() {
  try { await api.NewChat(); } catch (e) { addMessage("error", String(e)); return; }
  $("messages").replaceChildren();
  state.current = null;
  state.draft = "";
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
  closeSuggest();
  state.files = [];
  renderAttachments();
  state.turn = null;
  state.cmd = null;
  try {
    if (text.startsWith("/")) {
      await api.Command(text);
    } else {
      if (!state.current && !state.draft) { state.draft = text; refreshSessions(); }
      await api.Send(text, files);
    }
    setRunning(true);
  } catch (e) { addMessage("error", String(e)); }
}

function onEvent(e) {
  switch (e.kind) {
    case "token":
      if (!state.turn) state.turn = addMessage("assistant", "");
      appendAssistant(state.turn, e.text);
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
    case "command_out":
      onCommandOut(e.text);
      break;
    case "model":
      onModelEvent(e);
      break;
    case "ui":
      onUIEvent(e);
      break;
    case "error": {
      const m = addMessage("error", e.text);
      addSwitchModelButton(m);
      setRunning(false);
      break;
    }
    case "turn_end":
      state.current = e.sessionId || state.current;
      if (state.current) state.draft = "";
      state.turn = null;
      state.cmd = null;
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
    if (dir) { state.current = null; state.draft = ""; $("messages").replaceChildren(); await refreshWorkspace(); refreshSessions(); refreshModelBtn(); loadCommands(); }
  } catch (e) { addMessage("error", String(e)); }
}

let dragDepth = 0;
window.addEventListener("dragenter", () => { dragDepth++; $("drop").hidden = false; });
window.addEventListener("dragleave", () => { if (--dragDepth <= 0) { dragDepth = 0; $("drop").hidden = true; } });
window.addEventListener("drop", () => { dragDepth = 0; $("drop").hidden = true; });
window.addEventListener("dragover", (e) => e.preventDefault());

$("composer").addEventListener("submit", send);
$("input").addEventListener("keydown", (ev) => {
  if (suggestHandleKey(ev)) return;
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
