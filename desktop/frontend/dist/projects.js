"use strict";
// Projects modal: list workspace + installed workflows, agencies and
// components; run them (one Run button, same as `kdeps run`), add them to
// chat, and build them with forms. Relies on app.js globals (api, $, el) and
// projectsctl.js.

const pj = { list: [], cur: null, tab: "overview", view: null, kinds: null, logs: new Map(), progress: [] };
const PJ_LOG_MAX = 300;

function pjStatus(parent) {
  const s = el("div", "status");
  parent.appendChild(s);
  return (msg, isErr) => { s.textContent = msg; s.className = "status" + (isErr ? " err" : ""); };
}

function pjButton(label, onClick, cls) {
  const b = el("button", cls || "", label);
  b.type = "button";
  b.addEventListener("click", onClick);
  return b;
}

// pjField appends a labelled control row and returns the control.
function pjField(parent, label, control, help) {
  const row = el("div", "field");
  const lab = el("label", "", label);
  control.id = "pj-" + Math.random().toString(36).slice(2);
  lab.htmlFor = control.id;
  row.append(lab, control);
  if (help) row.appendChild(el("div", "help", help));
  parent.appendChild(row);
  return control;
}

function pjInput(value, type) {
  const i = el("input");
  i.type = type || "text";
  i.value = value == null ? "" : value;
  return i;
}

function pjTextarea(value, rows) {
  const t = el("textarea");
  t.value = value == null ? "" : value;
  t.rows = rows || 4;
  t.spellcheck = false;
  return t;
}

function pjSelect(options, value) {
  const s = el("select");
  for (const o of options) {
    const opt = el("option", "", o === "" ? "(none)" : o);
    opt.value = o;
    s.appendChild(opt);
  }
  s.value = value == null ? "" : value;
  return s;
}

async function openProjects() {
  $("projects").showModal();
  await refreshProjects();
}

async function refreshProjects(selectPath) {
  try { pj.list = (await api.Projects()) || []; } catch (e) { pj.list = []; $("pj-pane").replaceChildren(el("div", "status err", String(e))); }
  if (selectPath) pj.cur = selectPath;
  if (pj.cur && !pj.list.some((p) => p.path === pj.cur)) pj.cur = null;
  if (!pj.cur && pj.list.length) pj.cur = pj.list[0].path;
  renderProjectList();
  renderProjectPane();
}

// refreshProjectList reloads the list only, keeping the open pane (and its
// status line) as it is.
async function refreshProjectList() {
  try { pj.list = (await api.Projects()) || []; } catch (_) { return; }
  renderProjectList();
}

function curProject() { return pj.list.find((p) => p.path === pj.cur) || null; }

function renderProjectList() {
  const ul = $("pj-list");
  ul.replaceChildren();
  const groups = groupProjects(pj.list);
  for (const [label, items] of [["Workspace", groups.workspace], ["Installed", groups.installed]]) {
    if (!items.length) continue;
    ul.appendChild(el("li", "eyebrow", label));
    for (const p of items) {
      const li = el("li", "pj-item" + (p.path === pj.cur ? " active" : ""));
      li.tabIndex = 0;
      const top = el("div", "pj-top");
      top.append(el("span", "pj-name", p.name), el("span", "badge", p.kind));
      li.appendChild(top);
      const marks = [];
      if (p.error) marks.push("error");
      if (p.running) marks.push("running");
      if (p.inChat) marks.push("in chat");
      if (marks.length) li.appendChild(el("div", "hint", marks.join(" - ")));
      const pick = () => { pj.cur = p.path; renderProjectList(); renderProjectPane(); };
      li.addEventListener("click", pick);
      li.addEventListener("keydown", (ev) => { if (ev.key === "Enter") pick(); });
      ul.appendChild(li);
    }
  }
  if (!pj.list.length) ul.appendChild(el("li", "hint", "No workflows, agencies or components in this workspace yet."));
}

function renderProjectPane() {
  const pane = $("pj-pane");
  pane.replaceChildren();
  const p = curProject();
  for (const b of document.querySelectorAll("#pj-tabs button")) {
    b.setAttribute("aria-selected", String(b.dataset.tab === pj.tab));
    b.disabled = !p;
  }
  if (!p) {
    pane.appendChild(el("div", "hint", "Create a project with New, or open a workspace that has one."));
    return;
  }
  const render = { overview: renderOverview, run: renderRun, build: renderBuild }[pj.tab];
  Promise.resolve(render(pane, p)).catch((e) => pane.appendChild(el("div", "status err", String(e))));
}

// ---- Overview ----

function renderOverview(pane, p) {
  const head = el("div", "card");
  const top = el("div", "top");
  top.append(el("strong", "", p.name + (p.version ? " " + p.version : "")), el("span", "badge", p.kind));
  head.append(top);
  if (p.description) head.appendChild(el("div", "", p.description));
  head.appendChild(el("div", "hint pj-path", p.path));
  if (p.installed) head.appendChild(el("div", "hint", "Installed package - read-only. Run it or add it to chat."));
  if (p.error) head.appendChild(el("pre", "pj-err", p.error));
  pane.appendChild(head);

  const say = pjStatus(pane);

  // Use in chat
  pane.appendChild(el("div", "eyebrow", "Chat"));
  const chatRow = el("label", "toggle");
  const chat = el("input");
  chat.type = "checkbox";
  chat.checked = p.inChat;
  chat.disabled = !!p.error;
  chat.addEventListener("change", async () => {
    try {
      await api.SetInChat(p.path, chat.checked);
      say(chat.checked ? p.name + " is now a tool the chat agent can call." : p.name + " removed from chat tools.");
      await refreshProjectList();
    } catch (e) { chat.checked = !chat.checked; say(String(e), true); }
  });
  chatRow.append(chat, el("span", "", "Let the chat agent call " + p.name + " as a tool"));
  pane.appendChild(chatRow);

  // Manage
  pane.appendChild(el("div", "eyebrow", "Manage"));
  const row = el("div", "pj-actions");
  row.append(
    pjButton("Validate", async () => {
      const problems = (await api.ValidateProject(p.path)) || [];
      say(problems.length ? problems.join("\n") : "Valid.", problems.length > 0);
    }),
    pjButton("Open folder", () => api.RevealProject(p.path).catch((e) => say(String(e), true))),
  );
  if (!p.installed) {
    const del = pjButton("Delete", async () => {
      if (del.dataset.armed !== "1") { del.dataset.armed = "1"; del.textContent = "Delete folder " + p.dir + "?"; return; }
      try { await api.DeleteProject(p.path); pj.cur = null; await refreshProjects(); }
      catch (e) { say(String(e), true); }
    });
    row.appendChild(del);
  }
  pane.appendChild(row);
}

// ---- Run ----

function renderRun(pane, p) {
  return p.kind === "component" ? renderComponentRun(pane, p) : renderProcessRun(pane, p);
}

// renderProcessRun shows the environment and the single Run/Stop control of
// a workflow or agency. Run is `kdeps run <dir>` in the workspace, with the
// project's .env and the variables below added.
function renderProcessRun(pane, p) {
  pane.appendChild(el("div", "eyebrow", "Environment"));
  const envCard = el("div", "card pj-form");
  const dotenv = el("div", "hint", "");
  envCard.appendChild(dotenv);
  api.ProjectDotEnv(p.path).then((names) => {
    dotenv.textContent = names && names.length
      ? ".env in the project folder loads automatically: " + names.join(", ")
      : "A .env file in the project folder loads automatically on every run.";
  }).catch((e) => { dotenv.textContent = String(e); dotenv.className = "status err"; });
  const env = pjField(envCard, "variables", Object.assign(pjTextarea("", 4), { className: "mono" }),
    "NAME=value per line. These override .env. $PWD is the workspace folder. Import from file reads NAME=value lines from any text file.");
  env.placeholder = "DATA_DIR=$PWD/" + p.dir.split(/[\\/]/).pop();
  api.ProjectEnv(p.path).then((t) => { env.value = t || ""; }).catch(() => {});
  const envSay = pjStatus(envCard);
  const save = async () => {
    try { await api.SetProjectEnv(p.path, env.value); envSay("Saved."); return true; }
    catch (e) { envSay(String(e), true); return false; }
  };
  env.addEventListener("change", save);
  const name = pjInput("");
  name.placeholder = "NAME";
  name.setAttribute("aria-label", "Variable name");
  const value = pjInput("");
  value.placeholder = "value";
  value.setAttribute("aria-label", "Variable value");
  const add = el("div", "pj-actions pj-envadd");
  add.append(name, value, pjButton("Add", async () => {
    if (!name.value.trim()) { envSay("Enter a variable name.", true); return; }
    const line = name.value.trim() + "=" + value.value;
    env.value = env.value.trim() ? env.value.replace(/\s*$/, "") + "\n" + line : line;
    if (await save()) { name.value = ""; value.value = ""; name.focus(); }
  }), pjButton("Import from file", async () => {
    try {
      const text = await api.ImportEnvFile();
      if (!text) return;
      env.value = env.value.trim() ? env.value.replace(/\s*$/, "") + "\n" + text : text;
      await save();
    } catch (e) { envSay(String(e), true); }
  }));
  envCard.insertBefore(add, envCard.lastChild);
  pane.appendChild(envCard);

  pane.appendChild(el("div", "eyebrow", "Run"));
  const card = el("div", "card pj-form");
  card.appendChild(el("div", "hint", "Same as `kdeps run " + p.dir.split(/[\\/]/).pop() + "` from the workspace: an API or web server keeps running until you stop it; anything else runs once."));
  const say = pjStatus(card);
  const btn = pjButton(p.running ? "Stop" : "Run", async () => {
    btn.disabled = true;
    try {
      if (p.running) {
        await api.Stop(p.path);
      } else {
        if (!(await save())) { btn.disabled = false; return; }
        pj.logs.set(p.path, []);
        await api.Run(p.path);
      }
      await refreshProjects();
    } catch (e) { say(String(e), true); btn.disabled = false; }
  }, p.running ? "" : "primary");
  btn.disabled = !!p.error;
  card.insertBefore(btn, card.lastChild);
  if (p.running) {
    say(p.url ? "Running at " + p.url : "Running");
    for (const r of p.url ? p.routes || [] : []) {
      const line = el("div", "pj-curl");
      const code = el("code", "", curlFor(p.url, r));
      line.append(code, pjButton("Copy", () => navigator.clipboard.writeText(code.textContent).then(() => say("Copied."))));
      card.appendChild(line);
    }
  }
  const log = el("pre", "pj-log");
  log.id = "pj-log";
  log.textContent = (pj.logs.get(p.path) || []).join("\n");
  card.appendChild(log);
  pane.appendChild(card);
  log.scrollTop = log.scrollHeight;
  return say;
}

// renderComponentRun runs a component once with its declared inputs.
function renderComponentRun(pane, p) {
  const card = el("div", "card pj-form");
  const ctrls = {};
  for (const inp of p.inputs || []) {
    const c = inp.type === "boolean" ? Object.assign(el("input"), { type: "checkbox" }) : pjInput("", inp.type === "string" ? "text" : "number");
    ctrls[inp.name] = pjField(card, inp.name + (inp.required ? " *" : ""), c, inp.description || inp.type);
  }
  if (!(p.inputs || []).length) card.appendChild(el("div", "hint", "This component declares no inputs."));
  const progress = el("ul", "pj-progress");
  const out = el("pre", "pj-out");
  let say;
  const btn = pjButton("Run", async () => {
    const vals = {};
    for (const [k, c] of Object.entries(ctrls)) vals[k] = c.type === "checkbox" ? c.checked : c.value;
    let inputs;
    try { inputs = componentBody(p.inputs, vals); } catch (e) { say(e.message, true); return; }
    btn.disabled = true;
    progress.replaceChildren();
    out.textContent = "";
    pj.progress = progress;
    say("Running...");
    try {
      const res = await api.RunComponent(p.path, inputs);
      out.textContent = res.output || "(no output)";
      say((res.error ? "Failed: " + res.error : "Done") + " in " + res.durationMs + " ms", !!res.error);
    } catch (e) { say(String(e), true); }
    btn.disabled = false;
  }, "primary");
  btn.disabled = !!p.error;
  card.appendChild(btn);
  say = pjStatus(card);
  card.append(progress, out);
  pane.appendChild(card);
  return say;
}

// ---- Build ----

async function renderBuild(pane, p) {
  if (!pj.kinds) pj.kinds = (await api.ResourceKinds()) || [];
  const view = await api.OpenBuilder(p.path);
  pj.view = view;
  if (view.readOnly) pane.appendChild(el("div", "hint", p.installed
    ? "Installed package - read-only. Create a project in the workspace to build your own."
    : "Jinja2 manifest - edit it in a text editor (Open folder)."));

  pane.appendChild(el("div", "eyebrow", p.kind));
  renderManifestForm(pane, p, view);
  if (p.kind === "agency") {
    pane.appendChild(el("div", "hint", "An agency's agents are separate workflows. Build each one from its own entry in the list."));
    return;
  }
  pane.appendChild(el("div", "eyebrow", "Resources"));
  const list = el("div", "pj-resources");
  for (const r of view.resources || []) list.appendChild(resourceCard(p, view, r));
  pane.appendChild(list);
  if (!view.readOnly) {
    const add = el("div", "pj-actions");
    const kindSel = pjSelect(pj.kinds.map((k) => k.key), "chat");
    kindSel.setAttribute("aria-label", "Resource type");
    add.append(kindSel, pjButton("Add resource", () => {
      const r = { id: "", actionId: "", name: "", description: "", requires: [], kind: kindSel.value, action: {}, extra: "" };
      list.appendChild(resourceCard(p, view, r, true));
    }));
    pane.appendChild(add);
  }
}

function renderManifestForm(pane, p, view) {
  const m = view.manifest;
  const box = el("div", "card");
  const name = pjField(box, "name", pjInput(m.name));
  const desc = pjField(box, "description", pjTextarea(m.description, 2));
  const version = pjField(box, "version", pjInput(m.version));
  let target;
  if (p.kind === "agency") {
    target = pjField(box, "targetAgentId", pjInput(m.target), "The agent that receives requests first.");
  } else {
    const targets = [""].concat(resourceTargets(view.resources));
    if (m.target && !targets.includes(m.target)) targets.push(m.target);
    target = pjField(box, "targetActionId", pjSelect(targets, m.target), "The resource whose output is the response.");
  }
  let body;
  if (m.section) body = pjField(box, m.section, Object.assign(pjTextarea(m.body, 8), { className: "mono" }), m.section === "settings" ? "YAML: apiServer routes, agentSettings, ..." : "YAML: inputs the component accepts.");
  const say = pjStatus(box);
  if (!view.readOnly) {
    box.appendChild(pjButton("Save", async () => {
      try {
        const problems = (await api.SaveManifest(p.path, { name: name.value, description: desc.value, version: version.value,
          target: target.value, section: m.section, body: body ? body.value : "" })) || [];
        say(problems.length ? "Saved, with problems:\n" + problems.join("\n") : "Saved.", problems.length > 0);
        await refreshProjectList();
      } catch (e) { say(String(e), true); }
    }, "primary"));
  }
  pane.appendChild(box);
}

function resourceCard(p, view, r, open) {
  const card = el("details", "card pj-res");
  card.open = !!open;
  const sum = el("summary", "");
  sum.append(el("span", "pj-name", r.actionId || "(new resource)"), el("span", "badge", r.kind || "no action"), el("span", "hint", r.source || ""));
  card.appendChild(sum);
  if (r.readOnly && !r.kind) { card.appendChild(el("div", "hint", "Template file - edit it in a text editor.")); return card; }
  const body = el("div", "pj-form");
  card.appendChild(body);
  card.addEventListener("toggle", () => { if (card.open && !body.children.length) fillResourceForm(body, sum, p, view, r); }, { once: false });
  if (open) fillResourceForm(body, sum, p, view, r);
  return card;
}

function fillResourceForm(body, sum, p, view, r) {
  if (body.children.length) return;
  const ro = view.readOnly || r.readOnly;
  const actionId = pjField(body, "actionId", pjInput(r.actionId), "Unique id other resources use in requires and output().");
  const name = pjField(body, "name", pjInput(r.name));
  const desc = pjField(body, "description", pjTextarea(r.description, 2));

  // requires: checkboxes of the other resources
  const others = resourceTargets(view.resources).filter((a) => a !== r.actionId);
  const reqBox = el("div", "pj-checks");
  const reqs = new Set(r.requires || []);
  for (const a of new Set(others.concat([...reqs]))) {
    const lab = el("label", "toggle");
    const c = el("input");
    c.type = "checkbox";
    c.checked = reqs.has(a);
    c.value = a;
    lab.append(c, el("span", "", a));
    reqBox.appendChild(lab);
  }
  if (!reqBox.children.length) reqBox.appendChild(el("span", "hint", "No other resources yet."));
  const reqRow = el("div", "field");
  reqRow.append(el("label", "", "requires"), reqBox);
  body.appendChild(reqRow);

  const kind = pj.kinds.find((k) => k.key === r.kind) || null;
  body.appendChild(el("div", "eyebrow", r.kind || "action"));
  const fields = el("div", "pj-fields");
  body.appendChild(fields);
  const ctrls = new Map();
  const addField = (f) => {
    const v = r.action ? r.action[f.name] : undefined;
    let c;
    switch (controlFor(f)) {
      case "checkbox": c = el("input"); c.type = "checkbox"; c.checked = v === true; break;
      case "select": c = pjSelect([""].concat(f.enum || []), v); break;
      case "textarea": c = pjTextarea(v, f.type === "yaml" ? 4 : 5); if (f.type === "yaml") c.classList.add("mono"); break;
      case "number": c = pjInput(v, "text"); c.inputMode = "decimal"; break;
      default: c = pjInput(v);
    }
    ctrls.set(f.name, { f, c });
    pjField(fields, f.name, c, (f.type === "yaml" ? "YAML. " : "") + (f.description || ""));
  };
  for (const f of shownFields(kind, r.action)) addField(f);
  if (kind && !ro) {
    const more = addableFields(kind, r.action);
    if (more.length) {
      const pick = pjSelect([""].concat(more.map((f) => f.name)), "");
      pick.setAttribute("aria-label", "Add field");
      pick.addEventListener("change", () => {
        const f = kind.fields.find((x) => x.name === pick.value);
        if (f && !ctrls.has(f.name)) addField(f);
        pick.querySelector('option[value="' + pick.value + '"]').remove();
        pick.value = "";
      });
      const row = el("div", "field");
      row.append(el("label", "", "add field"), pick);
      body.appendChild(row);
    }
  }
  const extra = pjField(body, "advanced", pjTextarea(r.extra, 3), "YAML for anything else: validations, onError, loop, before, after, items, ...");
  extra.classList.add("mono");
  const say = pjStatus(body);

  if (ro) {
    for (const x of body.querySelectorAll("input, textarea, select")) x.disabled = true;
    return;
  }
  const actions = el("div", "pj-actions");
  actions.appendChild(pjButton("Save", async () => {
    const action = {};
    for (const [k, { f, c }] of ctrls) action[k] = formFieldValue(f, c.type === "checkbox" ? c.checked : c.value);
    const form = { id: r.id, actionId: actionId.value.trim(), name: name.value, description: desc.value,
      requires: [...reqBox.querySelectorAll("input:checked")].map((c) => c.value),
      kind: r.kind, action, extra: extra.value };
    try {
      const res = await api.SaveResource(p.path, form);
      Object.assign(r, res.resource);
      sum.firstChild.textContent = r.actionId;
      sum.lastChild.textContent = r.source;
      say(res.problems && res.problems.length ? "Saved, with problems:\n" + res.problems.join("\n") : "Saved.", !!(res.problems && res.problems.length));
      pj.view = await api.OpenBuilder(p.path);
      view.resources = pj.view.resources;
    } catch (e) { say(String(e), true); }
  }, "primary"));
  const del = pjButton("Delete", async () => {
    if (!r.id) { body.closest("details").remove(); return; }
    if (del.dataset.armed !== "1") { del.dataset.armed = "1"; del.textContent = "Delete " + (r.actionId || "resource") + "?"; return; }
    try { await api.DeleteResource(p.path, r.id); renderProjectPane(); } catch (e) { say(String(e), true); }
  });
  actions.appendChild(del);
  body.appendChild(actions);
}

// ---- New project ----

async function newProjectForm() {
  const pane = $("pj-pane");
  pane.replaceChildren(el("div", "eyebrow", "New project"));
  const box = el("div", "card");
  const name = pjField(box, "name", pjInput(""), "A folder with this name is created in the workspace.");
  const tpl = pjField(box, "template", pjSelect((await api.ProjectTemplates()) || [], "api-service"));
  const say = pjStatus(box);
  box.appendChild(pjButton("Create", async () => {
    try {
      const p = await api.CreateProject(tpl.value, name.value);
      pj.tab = "build";
      await refreshProjects(p.path);
    } catch (e) { say(String(e), true); }
  }, "primary"));
  pane.appendChild(box);
}

// ---- Events ----

function onProjectEvent(e) {
  if (e.kind === "run_progress") {
    if (pj.progress && pj.progress.appendChild) pj.progress.appendChild(el("li", e.error ? "err" : "", progressLabel(e)));
    return;
  }
  if (e.kind === "run_log") {
    const lines = trimLog((pj.logs.get(e.tool) || []).concat([e.text]), PJ_LOG_MAX);
    pj.logs.set(e.tool, lines);
    const log = $("pj-log");
    if (log && pj.cur === e.tool) { log.textContent = lines.join("\n"); log.scrollTop = log.scrollHeight; }
    return;
  }
  if (e.kind === "run_stopped" && e.error) {
    pj.logs.set(e.tool, trimLog((pj.logs.get(e.tool) || []).concat(["exited: " + e.text]), PJ_LOG_MAX));
  }
  // run_url and run_stopped change the Run control and the list marks.
  if ($("projects").open) refreshProjects();
}

function initProjects() {
  const nav = $("pj-tabs");
  for (const [id, label] of [["overview", "Overview"], ["run", "Run"], ["build", "Build"]]) {
    const b = el("button", "", label);
    b.type = "button";
    b.dataset.tab = id;
    b.setAttribute("role", "tab");
    b.addEventListener("click", () => { pj.tab = id; renderProjectPane(); });
    nav.appendChild(b);
  }
  $("open-projects").addEventListener("click", openProjects);
  $("projects-close").addEventListener("click", () => $("projects").close());
  $("pj-new").addEventListener("click", newProjectForm);
  $("pj-refresh").addEventListener("click", () => refreshProjects());
}

initProjects();
