/* Personal Assistant — kdeps web UI */

const CHAT_API   = '/api/v1/chat';
const HB_API     = '/api/v1/heartbeat';
const STORAGE_KEY = 'pa_sessions';

// ── State ────────────────────────────────────────────────────────────────────
let sessions   = {};   // { [id]: { id, name, messages: [] } }
let currentId  = null;
let isLoading  = false;

// ── DOM refs ─────────────────────────────────────────────────────────────────
const $messages    = document.getElementById('messages');
const $welcome     = document.getElementById('welcome');
const $sessionList = document.getElementById('session-list');
const $sessionDisp = document.getElementById('session-id-display');
const $form        = document.getElementById('chat-form');
const $input       = document.getElementById('msg-input');
const $sendBtn     = document.getElementById('send-btn');
const $btnNew      = document.getElementById('btn-new-session');
const $btnClear    = document.getElementById('btn-clear');
const $btnHB       = document.getElementById('btn-heartbeat');
const $hbBackdrop  = document.getElementById('hb-backdrop');
const $hbBody      = document.getElementById('hb-body');
const $hbClose     = document.getElementById('hb-close');

// ── Persistence ───────────────────────────────────────────────────────────────
function load() {
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (raw) sessions = JSON.parse(raw);
  } catch (_) {}
}
function save() {
  try { localStorage.setItem(STORAGE_KEY, JSON.stringify(sessions)); } catch (_) {}
}

// ── Session helpers ───────────────────────────────────────────────────────────
function genId() {
  return Date.now().toString(36) + Math.random().toString(36).slice(2, 7);
}

function newSession(name) {
  const id = genId();
  sessions[id] = { id, name: name || `Chat ${Object.keys(sessions).length + 1}`, messages: [] };
  save();
  return id;
}

function switchSession(id) {
  currentId = id;
  $sessionDisp.textContent = id;
  renderSessionList();
  renderMessages();
}

function sessionName(messages) {
  const first = messages.find(m => m.role === 'user');
  if (!first) return null;
  return first.content.length > 32 ? first.content.slice(0, 32) + '…' : first.content;
}

// ── Render session list ───────────────────────────────────────────────────────
function renderSessionList() {
  $sessionList.innerHTML = '';
  Object.values(sessions).reverse().forEach(s => {
    const el = document.createElement('div');
    el.className = 'session-item' + (s.id === currentId ? ' active' : '');
    el.innerHTML = `
      <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
        <path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"/>
      </svg>
      <span class="session-name">${esc(s.name)}</span>`;
    el.addEventListener('click', () => switchSession(s.id));
    $sessionList.appendChild(el);
  });
}

// ── Render messages ───────────────────────────────────────────────────────────
function renderMessages() {
  const session = sessions[currentId];
  if (!session) return;
  const msgs = session.messages;

  if (msgs.length === 0) {
    $welcome.style.display = '';
    // Remove all message rows
    [...$messages.querySelectorAll('.msg-row')].forEach(el => el.remove());
    return;
  }

  $welcome.style.display = 'none';
  const existing = [...$messages.querySelectorAll('.msg-row')];
  // Re-render all (simple approach)
  existing.forEach(el => el.remove());
  msgs.forEach(m => $messages.appendChild(buildMsgEl(m)));
  scrollBottom();
}

function buildMsgEl(msg) {
  const row = document.createElement('div');
  row.className = 'msg-row ' + msg.role + (msg.error ? ' error' : '');

  const avatar = document.createElement('div');
  avatar.className = 'msg-avatar';
  avatar.textContent = msg.role === 'user' ? 'You' : 'AI';

  const bubble = document.createElement('div');
  bubble.className = 'msg-bubble';
  bubble.innerHTML = formatContent(msg.content);

  row.appendChild(avatar);
  row.appendChild(bubble);
  return row;
}

function addTypingIndicator() {
  const row = document.createElement('div');
  row.className = 'msg-row assistant';
  row.id = 'typing';
  row.innerHTML = `
    <div class="msg-avatar">AI</div>
    <div class="msg-bubble"><div class="typing-dots"><span></span><span></span><span></span></div></div>`;
  $messages.appendChild(row);
  scrollBottom();
}
function removeTypingIndicator() {
  const el = document.getElementById('typing');
  if (el) el.remove();
}

// ── Content formatter ─────────────────────────────────────────────────────────
function esc(s) {
  return String(s)
    .replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;');
}

function formatContent(raw) {
  if (typeof raw !== 'string') raw = JSON.stringify(raw, null, 2);

  // Escape HTML first
  let s = raw.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;');

  // Fenced code blocks
  s = s.replace(/```(\w*)\n([\s\S]*?)```/g, (_, lang, code) =>
    `<pre><code>${code.trim()}</code></pre>`);

  // Inline code
  s = s.replace(/`([^`\n]+)`/g, '<code>$1</code>');

  // Bold
  s = s.replace(/\*\*([^*]+)\*\*/g, '<strong>$1</strong>');

  // Italic
  s = s.replace(/\*([^*\n]+)\*/g, '<em>$1</em>');

  // Newlines → <br> (outside pre blocks)
  s = s.replace(/\n/g, '<br>');

  return s;
}

function scrollBottom() {
  $messages.scrollTop = $messages.scrollHeight;
}

// ── Send message ──────────────────────────────────────────────────────────────
async function sendMessage(content) {
  if (!content || isLoading) return;
  if (!currentId) currentId = newSession();

  const session = sessions[currentId];
  session.messages.push({ role: 'user', content });
  if (!session.name || session.name.startsWith('Chat ')) {
    session.name = sessionName(session.messages) || session.name;
  }
  save();
  $welcome.style.display = 'none';
  $messages.appendChild(buildMsgEl({ role: 'user', content }));
  addTypingIndicator();
  scrollBottom();

  isLoading = true;
  $sendBtn.disabled = true;

  try {
    const res = await fetch(CHAT_API, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ message: content, session_id: currentId }),
    });

    const json = await res.json();
    removeTypingIndicator();

    // Gateway returns { success: true, data: "<text>" }
    let reply;
    if (json.success && json.data !== undefined) {
      reply = typeof json.data === 'string' ? json.data : JSON.stringify(json.data, null, 2);
    } else {
      reply = json.error?.message || json.message || 'Something went wrong.';
    }

    session.messages.push({ role: 'assistant', content: reply });
    save();
    renderSessionList();
    $messages.appendChild(buildMsgEl({ role: 'assistant', content: reply }));
    scrollBottom();

  } catch (err) {
    removeTypingIndicator();
    const errMsg = 'Could not reach the API. Is kdeps running?';
    session.messages.push({ role: 'assistant', content: errMsg, error: true });
    save();
    $messages.appendChild(buildMsgEl({ role: 'assistant', content: errMsg, error: true }));
    scrollBottom();
  }

  isLoading = false;
  $sendBtn.disabled = $input.value.trim() === '';
}

// ── Heartbeat ─────────────────────────────────────────────────────────────────
async function runHeartbeat() {
  $hbBody.innerHTML = '<div class="spinner"></div><p>Running heartbeat check…</p>';
  $hbBackdrop.hidden = false;
  $btnHB.classList.add('loading');

  try {
    const res = await fetch(HB_API, { method: 'POST' });
    const json = await res.json();
    renderHeartbeatResult(json);
  } catch (err) {
    $hbBody.innerHTML = `<p style="color:var(--danger)">Failed to reach heartbeat endpoint. Is kdeps running?</p>`;
  }

  $btnHB.classList.remove('loading');
}

function renderHeartbeatResult(json) {
  if (!json.success) {
    $hbBody.innerHTML = `<p style="color:var(--danger)">${esc(JSON.stringify(json))}</p>`;
    return;
  }

  const d = json.data || {};
  let assessment = d.assessment;
  let actionsRaw = d.actions_taken || '';

  // assessment may be a JSON string from the brain
  if (typeof assessment === 'string') {
    try { assessment = JSON.parse(assessment); } catch (_) {}
  }

  let html = '';

  if (assessment && typeof assessment === 'object') {
    // Summary
    if (assessment.summary) {
      html += `<div class="hb-section"><h4>Summary</h4><p>${esc(assessment.summary)}</p></div>`;
    }

    // Tasks to act on
    if (Array.isArray(assessment.needs_action) && assessment.needs_action.length) {
      html += `<div class="hb-section"><h4>Acting on</h4>`;
      assessment.needs_action.forEach(t => {
        html += `<div class="hb-task">
          <span class="hb-badge act">ACT</span>
          <div><div class="task-name">${esc(t.task || '')}</div>
          <div class="task-reason">${esc(t.reason || '')} — ${esc(t.action || '')}</div></div>
        </div>`;
      });
      html += `</div>`;
    }

    // Deferred
    if (Array.isArray(assessment.deferred) && assessment.deferred.length) {
      html += `<div class="hb-section"><h4>Deferred</h4>`;
      assessment.deferred.forEach(t => {
        html += `<div class="hb-task">
          <span class="hb-badge defer">LATER</span>
          <div class="task-name">${esc(t)}</div>
        </div>`;
      });
      html += `</div>`;
    }
  } else if (assessment) {
    html += `<div class="hb-section"><h4>Assessment</h4><p>${esc(String(assessment))}</p></div>`;
  }

  // Raw actions output
  if (actionsRaw) {
    html += `<div class="hb-section"><h4>Actions output</h4>
      <div class="hb-actions-raw">${esc(actionsRaw)}</div></div>`;
  }

  if (!html) html = '<p>No heartbeat data returned.</p>';
  $hbBody.innerHTML = html;
}

// ── Event listeners ───────────────────────────────────────────────────────────
$form.addEventListener('submit', e => {
  e.preventDefault();
  const val = $input.value.trim();
  if (!val || isLoading) return;
  $input.value = '';
  $input.style.height = 'auto';
  $sendBtn.disabled = true;
  sendMessage(val);
});

$input.addEventListener('input', () => {
  $sendBtn.disabled = $input.value.trim() === '' || isLoading;
  $input.style.height = 'auto';
  $input.style.height = Math.min($input.scrollHeight, 180) + 'px';
});

$input.addEventListener('keydown', e => {
  if (e.key === 'Enter' && !e.shiftKey) {
    e.preventDefault();
    if (!$sendBtn.disabled) $form.dispatchEvent(new Event('submit'));
  }
});

$btnNew.addEventListener('click', () => {
  const id = newSession();
  switchSession(id);
});

$btnClear.addEventListener('click', () => {
  if (!currentId) return;
  sessions[currentId].messages = [];
  save();
  renderMessages();
});

$btnHB.addEventListener('click', () => {
  if (!$btnHB.classList.contains('loading')) runHeartbeat();
});
$hbClose.addEventListener('click', () => { $hbBackdrop.hidden = true; });
$hbBackdrop.addEventListener('click', e => {
  if (e.target === $hbBackdrop) $hbBackdrop.hidden = true;
});

// Suggestion chips
document.querySelectorAll('.suggestion').forEach(btn => {
  btn.addEventListener('click', () => {
    const msg = btn.dataset.msg;
    if (msg) sendMessage(msg);
  });
});

// ── Boot ──────────────────────────────────────────────────────────────────────
load();

// Restore or create initial session
const ids = Object.keys(sessions);
if (ids.length === 0) {
  currentId = newSession('Chat 1');
} else {
  currentId = ids[ids.length - 1];
}

$sessionDisp.textContent = currentId;
renderSessionList();
renderMessages();
