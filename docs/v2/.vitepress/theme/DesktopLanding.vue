<!--
  Copyright 2026 Kdeps, KvK 94834768
  Licensed under the Apache License, Version 2.0
-->
<script setup lang="ts">
import { ref } from 'vue'

const releases = 'https://github.com/kdeps/kdeps/releases/latest'

const installs = [
  { os: 'macOS', note: 'Apple Silicon and Intel, macOS 11+', cmd: 'brew install --cask kdeps/tap/kdeps-desktop', file: '.dmg' },
  { os: 'Windows', note: 'x64, WebView2 (ships with Windows 11)', cmd: 'scoop bucket add kdeps https://github.com/kdeps/scoop-bucket\nscoop install kdeps-desktop', file: '.zip' },
  { os: 'Linux', note: 'x86_64, GTK 3 + WebKitGTK 4.1', cmd: 'sudo apt install ./kdeps-desktop_<version>_linux_amd64.deb\n# also .rpm, Arch .pkg.tar.zst and .tar.gz', file: '.deb / .rpm / .tar.gz' },
]

const copied = ref('')
function copy(text: string) {
  navigator.clipboard?.writeText(text)
  copied.value = text
  setTimeout(() => { if (copied.value === text) copied.value = '' }, 1500)
}

const local = ['llamafile', 'GGUF', 'Ollama']
const cloud = [
  'OpenAI-compatible', 'Anthropic', 'Google', 'AWS Bedrock', 'Cohere', 'Hugging Face',
  'Cloudflare', 'IBM watsonx', 'ERNIE', 'Maritaca', 'Microsoft 365 Copilot',
]

const features = [
  { title: 'Drop a file, get an answer', body: 'Drag files or folders onto the window. The agent works out what each one is, reads it with the right tool - text, PDF, DOCX, CSV, code - and tells you what you can do with it.' },
  { title: 'An agent, not a chat box', body: 'The same loop as the terminal: shell, file edits, web search, SQL, memory, goals and a judge panel. Your workflows and agencies show up as tools it can call.' },
  { title: 'You approve every action', body: 'In ask mode each tool call and every path outside the workspace opens an approval: allow once, allow always, or deny.' },
  { title: 'Workspaces with memory', body: 'A workspace is a folder. History, search and persistent memory are kept per folder, and file tools never leave it.' },
  { title: 'Every setting, no YAML', body: 'Settings are generated from the config schema: dropdowns, sliders and suggestions for every field, plus harness sections, custom instructions and memories.' },
  { title: 'Build and run workflows', body: 'Create workflows and components from a template, add resources with forms, run them with one button, or hand them to the chat agent as tools.' },
]

const rows = [
  { label: 'Your prompts and files leave your boundary', hosted: 'Always', kdeps: 'Never, with a local model or your own LLM server' },
  { label: 'Works offline / air-gapped', hosted: 'No', kdeps: 'Yes - the UI loads nothing from a CDN' },
  { label: 'Edits files and runs commands for you', hosted: 'In a remote sandbox', kdeps: 'In your workspace, with approval' },
  { label: 'Choice of model', hosted: "The vendor's", kdeps: '300+ local builds, Ollama, a server you run, or a cloud key' },
  { label: 'Price', hosted: 'Subscription or per token', kdeps: 'Free, Apache 2.0' },
]
</script>

<template>
  <div class="dl">
    <!-- Hero -->
    <section class="hero">
      <div class="container">
        <p class="eyebrow">kdeps desktop</p>
        <h1>Your AI agent,<br>on your desk.</h1>
        <p class="lede">
          The kdeps agent in a native window for macOS, Windows and Linux. Run a local model and nothing
          leaves your machine, or point Settings at an <a href="/llm-server/">LLM server</a> you run.
          Drop in files, approve its actions, keep the history.
          The boundary is <a href="/start/data-sovereignty">data sovereignty</a>.
        </p>
        <div class="ctas">
          <a class="btn primary" :href="releases">Download for macOS</a>
          <a class="btn" :href="releases">Windows</a>
          <a class="btn" :href="releases">Linux</a>
        </div>
        <button class="cmd" type="button" @click="copy(installs[0].cmd)" :aria-label="'Copy ' + installs[0].cmd">
          <span class="prompt">$</span> {{ installs[0].cmd }}
          <span class="copy">{{ copied === installs[0].cmd ? 'copied' : 'copy' }}</span>
        </button>

        <!-- App window mock -->
        <div class="window" aria-label="kdeps desktop window">
          <div class="titlebar"><span></span><span></span><span></span><b>kdeps</b></div>
          <div class="app">
            <aside class="side">
              <div class="brand"><i>k</i> kdeps</div>
              <div class="new">New chat</div>
              <div class="search">Search chats</div>
              <div class="hist active">Quarterly report review</div>
              <div class="hist">Fix flaky test</div>
              <div class="hist">Summarize meeting notes</div>
              <div class="ws">~/work/finance</div>
            </aside>
            <main class="chat">
              <div class="model"><span>lfm2.5</span><em>gguf - local</em></div>
              <div class="msg user">Analyze: q3-report.pdf, budget.csv</div>
              <div class="tool"><b>read_file</b> q3-report.pdf <em>ok</em></div>
              <div class="tool"><b>load_document</b> budget.csv <em>ok</em></div>
              <div class="msg bot">
                <strong>q3-report.pdf</strong> - 14-page quarterly report. Revenue up 8%, two cost lines over plan.<br>
                <strong>budget.csv</strong> - 212 rows; the same two lines are 11% and 6% over.<br>
                <span class="dim">I can draft a variance summary or add the missing Q4 forecast column.</span>
              </div>
              <div class="composer"><span>+</span><p>Message kdeps - drop files anywhere</p><b>Send</b></div>
            </main>
          </div>
        </div>
      </div>
    </section>

    <!-- Models -->
    <section class="band">
      <div class="container">
        <p class="eyebrow">models</p>
        <h2>Every model kdeps knows</h2>
        <p class="sub">The picker lists what <code>/model list</code> shows. Local models download on first use, with progress in the chat.</p>
        <div class="stats">
          <div><b>156</b><span>GGUF builds</span></div>
          <div><b>159</b><span>llamafiles</span></div>
          <div><b>11</b><span>cloud backends</span></div>
        </div>
        <div class="chips">
          <span v-for="m in local" :key="m" class="chip on">{{ m }}</span>
          <span v-for="m in cloud" :key="m" class="chip">{{ m }}</span>
        </div>
      </div>
    </section>

    <!-- Features -->
    <section class="band">
      <div class="container">
        <p class="eyebrow">why kdeps desktop</p>
        <h2>More than a chat window</h2>
        <p class="sub">It is the same agent loop the terminal runs, so tools, memory, sessions, goals and judges behave the same.</p>
        <div class="grid">
          <div v-for="f in features" :key="f.title" class="card">
            <h3>{{ f.title }}</h3>
            <p>{{ f.body }}</p>
          </div>
        </div>
      </div>
    </section>

    <!-- Comparison -->
    <section class="band">
      <div class="container">
        <p class="eyebrow">local vs hosted</p>
        <h2>Own the agent, not a subscription</h2>
        <p class="sub">What changes when the agent runs on your machine.</p>
        <div class="table-wrap">
          <table>
            <thead><tr><th></th><th>Hosted AI chat</th><th>kdeps desktop</th></tr></thead>
            <tbody>
              <tr v-for="r in rows" :key="r.label">
                <td>{{ r.label }}</td><td>{{ r.hosted }}</td><td>{{ r.kdeps }}</td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </section>

    <!-- Steps -->
    <section class="band">
      <div class="container">
        <p class="eyebrow">getting started</p>
        <h2>Three steps</h2>
        <p class="sub">No CLI, no YAML, no account.</p>
        <div class="steps">
          <div class="step"><div class="num">01</div><h3>Install</h3><p>Homebrew, Scoop, a Linux package, or the file from the release page.</p></div>
          <div class="arrow">-></div>
          <div class="step"><div class="num">02</div><h3>Pick a model</h3><p>Click the model name. Choose a local build to stay offline, set the base URL of an LLM server you run, or pick a cloud model and paste its key in Settings.</p></div>
          <div class="arrow">-></div>
          <div class="step"><div class="num">03</div><h3>Ask or drop a file</h3><p>Type a question, or drop files on the window and the agent starts analyzing them.</p></div>
        </div>
      </div>
    </section>

    <!-- Value strip -->
    <section class="band">
      <div class="container strip">
        <div><b>0</b><span>requests to a CDN - the UI is embedded in the binary</span></div>
        <div><b>3</b><span>operating systems, one Go codebase</span></div>
        <div><b>No</b><span>Electron - the system WebView</span></div>
        <div><b>100%</b><span>open source, Apache 2.0</span></div>
      </div>
    </section>

    <!-- Download -->
    <section class="band" id="download">
      <div class="container">
        <p class="eyebrow">download</p>
        <h2>Install kdeps desktop</h2>
        <p class="sub">Standalone - the <code>kdeps</code> CLI is not needed. Every release attaches the files with a <code>.sha256</code> beside each.</p>
        <div class="grid three">
          <div v-for="i in installs" :key="i.os" class="card os">
            <h3>{{ i.os }}</h3>
            <p class="note">{{ i.note }}</p>
            <button class="code" type="button" @click="copy(i.cmd)" :aria-label="'Copy install command for ' + i.os">
              <pre>{{ i.cmd }}</pre><span class="copy">{{ copied === i.cmd ? 'copied' : 'copy' }}</span>
            </button>
            <a class="link" :href="releases">Download {{ i.file }} -></a>
          </div>
        </div>
        <p class="fine">
          macOS builds are ad-hoc signed and Windows builds are unsigned, so the first launch needs right-click, Open
          (macOS) or More info, Run anyway (Windows). <a href="/agent/desktop">Full install and build guide -></a>
        </p>
      </div>
    </section>

    <!-- Final CTA -->
    <section class="band final">
      <div class="container">
        <h2>Stop renting your AI. Run it.</h2>
        <p class="sub">A local model or your own server. Your files. Your machine.</p>
        <div class="ctas center">
          <a class="btn primary" :href="releases">Download kdeps desktop</a>
          <a class="btn" href="/agent/desktop">Read the docs</a>
        </div>
      </div>
    </section>
  </div>
</template>

<style scoped>
.dl { --hair: rgba(255, 255, 255, 0.08); }
.dl * { min-width: 0; }
.container { max-width: 1040px; margin: 0 auto; }
section { padding: 72px 24px; }
.band { border-top: 1px solid rgba(255, 255, 255, 0.06); }

.eyebrow {
  font-family: var(--vp-font-family-mono); font-size: 11px; font-weight: 600;
  letter-spacing: 0.12em; text-transform: uppercase; color: var(--vp-c-brand-1); margin: 0 0 10px;
}
h1 { font-size: 56px; line-height: 1.05; font-weight: 700; letter-spacing: -0.03em; margin: 0 0 18px; color: var(--vp-c-text-1); }
h2 { font-size: 36px; line-height: 1.15; font-weight: 700; letter-spacing: -0.02em; margin: 0 0 8px; color: var(--vp-c-text-1); border: 0; padding: 0; }
h3 { font-size: 17px; font-weight: 600; margin: 0 0 8px; color: var(--vp-c-text-1); }
.lede { font-size: 18px; line-height: 1.6; color: var(--vp-c-text-2); max-width: 640px; margin: 0 0 28px; }
.sub { font-size: 16px; color: var(--vp-c-text-2); margin: 0 0 32px; }
code { font-family: var(--vp-font-family-mono); font-size: 12px; color: #00E5FF; }

.hero { padding-top: 64px; }
.ctas { display: flex; flex-wrap: wrap; gap: 10px; margin-bottom: 16px; }
.ctas.center { justify-content: center; }
.btn {
  display: inline-block; padding: 10px 18px; border-radius: 2px; font-size: 14px; font-weight: 600;
  border: 1px solid var(--hair); color: var(--vp-c-text-1); text-decoration: none;
  transition: border-color 0.15s ease, transform 0.15s ease, background 0.15s ease;
}
.btn:hover { border-color: rgba(0, 229, 255, 0.5); transform: translateY(-2px); }
.btn.primary { background: #00E5FF; border-color: #00E5FF; color: #061016; }
.btn.primary:hover { background: #33ecff; }
.btn:focus-visible, .cmd:focus-visible, .code:focus-visible, .link:focus-visible { outline: 2px solid #00E5FF; outline-offset: 2px; }

.cmd, .code {
  display: flex; align-items: center; gap: 10px; width: 100%; text-align: left; cursor: pointer;
  font-family: var(--vp-font-family-mono); font-size: 13px; color: var(--vp-c-text-1);
  background: rgba(0, 0, 0, 0.2); border: 1px solid var(--hair); border-radius: 2px; padding: 10px 14px;
  transition: border-color 0.15s ease;
}
.cmd { max-width: 560px; margin-bottom: 48px; overflow-wrap: anywhere; }
.cmd:hover, .code:hover { border-color: rgba(0, 229, 255, 0.4); }
.prompt { flex-shrink: 0; color: var(--vp-c-text-3); }
.copy { flex-shrink: 0; white-space: nowrap; margin-left: auto; font-size: 11px; color: var(--vp-c-text-3); text-transform: uppercase; letter-spacing: 0.08em; }

/* App window mock */
.window { border: 1px solid var(--hair); border-radius: 2px; overflow: hidden; background: #0b0f14; box-shadow: 0 30px 80px rgba(0, 229, 255, 0.06); }
.titlebar { display: flex; align-items: center; gap: 6px; padding: 9px 12px; border-bottom: 1px solid var(--hair); background: #0e131a; }
.titlebar span { width: 10px; height: 10px; border-radius: 50%; background: rgba(255, 255, 255, 0.12); }
.titlebar b { margin: 0 auto; font-size: 12px; font-weight: 500; color: var(--vp-c-text-3); transform: translateX(-24px); }
.app { display: grid; grid-template-columns: 220px minmax(0, 1fr); min-height: 380px; }
.side { border-right: 1px solid var(--hair); padding: 14px; display: flex; flex-direction: column; gap: 8px; font-size: 13px; color: var(--vp-c-text-2); }
.brand { font-weight: 600; color: var(--vp-c-text-1); }
.brand i { font-style: normal; color: #00E5FF; margin-right: 4px; }
.new { border: 1px solid rgba(0, 229, 255, 0.6); color: #00E5FF; text-align: center; padding: 6px; border-radius: 2px; }
.search { border: 1px solid var(--hair); padding: 6px 8px; border-radius: 2px; color: var(--vp-c-text-3); }
.hist { padding: 6px 8px; border-radius: 2px; }
.hist.active { background: rgba(255, 255, 255, 0.05); color: var(--vp-c-text-1); }
.ws { margin-top: auto; font-family: var(--vp-font-family-mono); font-size: 11px; color: var(--vp-c-text-3); border: 1px solid var(--hair); padding: 6px 8px; border-radius: 2px; }
.chat { padding: 14px 18px; display: flex; flex-direction: column; gap: 10px; font-size: 13px; }
.model { display: flex; gap: 8px; align-items: center; }
.model span { font-family: var(--vp-font-family-mono); border: 1px solid var(--hair); padding: 3px 8px; border-radius: 2px; color: var(--vp-c-text-1); }
.model em { font-style: normal; color: var(--vp-c-text-3); font-size: 12px; }
.msg { padding: 10px 12px; border-radius: 2px; line-height: 1.6; overflow-wrap: anywhere; }
.msg.user { align-self: flex-end; background: rgba(0, 229, 255, 0.08); border: 1px solid rgba(0, 229, 255, 0.2); color: var(--vp-c-text-1); }
.msg.bot { color: var(--vp-c-text-1); border: 1px solid var(--hair); }
.dim { color: var(--vp-c-text-2); }
.tool { font-family: var(--vp-font-family-mono); font-size: 12px; color: var(--vp-c-text-2); border-left: 2px solid rgba(0, 229, 255, 0.4); padding: 2px 10px; }
.tool b { color: #00E5FF; font-weight: 500; margin-right: 6px; }
.tool em { font-style: normal; color: var(--vp-c-text-3); margin-left: 6px; }
.composer { margin-top: auto; display: flex; align-items: center; gap: 10px; border: 1px solid var(--hair); border-radius: 2px; padding: 8px 10px; color: var(--vp-c-text-3); }
.composer p { margin: 0; flex: 1; }
.composer b { color: #00E5FF; border: 1px solid rgba(0, 229, 255, 0.5); padding: 3px 10px; border-radius: 2px; font-weight: 500; }

/* Models */
.stats { display: grid; grid-template-columns: repeat(3, 1fr); gap: 12px; margin-bottom: 20px; }
.stats div, .strip div { border: 1px solid var(--hair); border-radius: 2px; padding: 18px; }
.stats b, .strip b { display: block; font-size: 32px; font-weight: 700; letter-spacing: -0.02em; color: #00E5FF; }
.stats span, .strip span { font-size: 13px; color: var(--vp-c-text-2); }
.chips { display: flex; flex-wrap: wrap; gap: 8px; }
.chip { font-family: var(--vp-font-family-mono); font-size: 12px; padding: 5px 10px; border-radius: 8px; border: 1px solid var(--hair); color: var(--vp-c-text-2); cursor: default; }
.chip.on { border-color: rgba(0, 229, 255, 0.4); color: #00E5FF; }

/* Cards */
.grid { display: grid; grid-template-columns: repeat(3, 1fr); gap: 12px; }
.card { border: 1px solid var(--hair); border-radius: 2px; padding: 20px; transition: border-color 0.15s ease, transform 0.15s ease; }
.card:hover { border-color: rgba(0, 229, 255, 0.3); transform: translateY(-2px); }
.card p { font-size: 14px; line-height: 1.65; color: var(--vp-c-text-2); margin: 0; }
.card.os { display: flex; flex-direction: column; gap: 12px; }
.card.os .note { font-size: 13px; color: var(--vp-c-text-3); }
.code pre { margin: 0; white-space: pre-wrap; word-break: break-all; font-size: 12px; line-height: 1.6; flex: 1; }
.link { margin-top: auto; font-size: 13px; color: var(--vp-c-brand-1); text-decoration: none; }
.link:hover { text-decoration: underline; }
.fine { font-size: 13px; color: var(--vp-c-text-3); margin: 20px 0 0; }
.fine a { color: var(--vp-c-brand-1); }

/* Table */
.table-wrap { overflow-x: auto; }
table { width: 100%; border-collapse: collapse; border: 1px solid rgba(255, 255, 255, 0.07); display: table; margin: 0; }
th { background: rgba(255, 255, 255, 0.05); font-size: 11px; letter-spacing: 0.07em; text-transform: uppercase; color: var(--vp-c-text-2); padding: 12px 18px; text-align: left; }
th:last-child { color: var(--vp-c-text-1); border-top: 2px solid rgba(0, 229, 255, 0.4); background: rgba(0, 229, 255, 0.04); }
td { padding: 14px 18px; font-size: 14px; color: var(--vp-c-text-2); border: 0; border-bottom: 1px solid rgba(255, 255, 255, 0.05); }
td:last-child { color: var(--vp-c-text-1); font-weight: 500; }
tr { background: transparent !important; border: 0; }

/* Steps */
.steps { display: flex; gap: 16px; align-items: flex-start; }
.step { flex: 1; }
.step p { font-size: 14px; line-height: 1.65; color: var(--vp-c-text-2); margin: 0; }
.num {
  width: 32px; height: 32px; border-radius: 50%; display: flex; align-items: center; justify-content: center; margin-bottom: 12px;
  background: rgba(0, 229, 255, 0.08); border: 1px solid rgba(0, 229, 255, 0.25);
  font-family: var(--vp-font-family-mono); font-size: 11px; font-weight: 600; color: var(--vp-c-brand-1);
}
.arrow { font-family: var(--vp-font-family-mono); color: rgba(255, 255, 255, 0.2); padding-top: 6px; }

.strip { display: grid; grid-template-columns: repeat(4, 1fr); gap: 12px; }
.final { text-align: center; }
.final .sub { margin-bottom: 24px; }

@media (max-width: 860px) {
  h1 { font-size: 40px; }
  h2 { font-size: 28px; }
  .grid, .grid.three, .strip { grid-template-columns: minmax(0, 1fr); }
  .app { grid-template-columns: minmax(0, 1fr); }
  .side { display: none; }
  .steps { flex-direction: column; }
  .arrow { transform: rotate(90deg); align-self: center; }
}
@media (max-width: 768px) {
  section { padding: 48px 16px; }
  .stats { grid-template-columns: minmax(0, 1fr); }
}
</style>
