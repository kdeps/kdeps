<!--
  Copyright 2026 Kdeps, KvK 94834768
  Licensed under the Apache License, Version 2.0
-->
<script setup>
import { ref } from 'vue'

const active = ref(0)

const K = (v) => `<span class="k">${v}</span>`
const S = (v) => `<span class="s">${v}</span>`
const N = (v) => `<span class="n">${v}</span>`
const O = (v) => `<span class="o">${v}</span>`

const T = (v) => `<span class="t">${v}</span>`
const D = (v) => `<span class="d">${v}</span>`
const P = (v) => `<span class="pr">${v}</span>`
const R = (v) => `<span class="re">${v}</span>`

const files = [
  {
    name: 'agent REPL',
    html: [
      `${P('$')} kdeps`,
      ``,
      `${D('kdeps v2.x  |  agent loop')}`,
      `${D('Model: llama3.2 (llamafile, offline)  |  /help for commands')}`,
      ``,
      `${T('>')} find the failing tests in ./api and suggest a fix`,
      ``,
      `${R('Ran go test ./api/... - 2 failures in handler_test.go.')}`,
      `${R('Both assert 200, but the router now returns 204 for an')}`,
      `${R('empty body. Update the expected status on lines 41 and 58.')}`,
      ``,
      `${T('>')} /model claude-sonnet`,
      `${D('Switched to claude-sonnet (Anthropic)')}`,
      ``,
      `${T('>')}`,
    ].join('\n'),
  },
  {
    name: 'workflow.yaml',
    html: [
      `${K('apiVersion')}${O(': ')}${S('kdeps.io/v1')}`,
      `${K('kind')}${O(': ')}${S('Workflow')}`,
      ``,
      `${K('metadata')}${O(':')}`,
      `  ${K('name')}${O(': ')}${S('chat-api')}`,
      `  ${K('version')}${O(': ')}${S('"1.0.0"')}`,
      `  ${K('targetActionId')}${O(': ')}${S('reply')}   ${D('# its output is the HTTP response')}`,
      ``,
      `${K('settings')}${O(':')}`,
      `  ${K('apiServer')}${O(':')}`,
      `    ${K('portNum')}${O(': ')}${N('16395')}`,
      `    ${K('routes')}${O(':')}`,
      `      ${O('- ')}${K('path')}${O(': ')}${S('/api/v1/chat')}`,
      `        ${K('methods')}${O(': ')}${S('[POST]')}`,
    ].join('\n'),
  },
  {
    name: 'resources/chat.yaml',
    html: [
      `${K('actionId')}${O(': ')}${S('chat')}`,
      `${K('name')}${O(': ')}${S('LLM Chat')}`,
      `${K('validations')}${O(':')}`,
      `  ${K('check')}${O(':')}`,
      `    ${O('- ')}${S('get(\'q\') != \'\'')}       ${D('# 400 if the body has no "q"')}`,
      `  ${K('error')}${O(':')}`,
      `    ${K('code')}${O(': ')}${N('400')}`,
      `    ${K('message')}${O(': ')}${S('"\'q\' is required"')}`,
      `${K('chat')}${O(':')}`,
      `  ${K('model')}${O(': ')}${S('llama3.2:1b')}     ${D('# local, no API key')}`,
      `  ${K('prompt')}${O(': ')}${S('"{{ get(\'q\') }}"')}`,
      `  ${K('timeout')}${O(': ')}${S('60s')}`,
    ].join('\n'),
  },
  {
    name: 'resources/reply.yaml',
    html: [
      `${K('actionId')}${O(': ')}${S('reply')}`,
      `${K('name')}${O(': ')}${S('API Response')}`,
      `${K('requires')}${O(': ')}${S('[chat]')}       ${D('# runs after chat')}`,
      `${K('apiResponse')}${O(':')}`,
      `  ${K('success')}${O(': ')}${S('true')}`,
      `  ${K('response')}${O(':')}`,
      `    ${K('answer')}${O(': ')}${S('get(\'chat\').message.content')}`,
      '', '', '', '', '', '',
    ].join('\n'),
  },
]
</script>

<template>
  <div class="hero-code-section">
    <div class="hero-code-container">
      <div class="hero-window">
        <div class="titlebar">
          <div class="dots">
            <span class="r"></span><span class="y"></span><span class="g"></span>
          </div>
          <div class="tabs">
            <button
              v-for="(f, i) in files"
              :key="i"
              :class="['tab', { active: active === i }]"
              @click="active = i"
            >{{ f.name }}</button>
          </div>
        </div>

        <pre class="code-body"><code v-html="files[active].html"></code></pre>

        <div class="terminal">
          <div class="tl"><span class="p">$</span><span class="c">export KDEPS_API_AUTH_TOKEN=dev-token</span></div>
          <div class="tl"><span class="p">$</span><span class="c">kdeps run .</span></div>
          <div class="tl dim">Listening on :16395</div>
          <div class="tl">&nbsp;</div>
          <div class="tl"><span class="p">$</span><span class="c">curl -s -X POST localhost:16395/api/v1/chat \</span></div>
          <div class="tl"><span class="pad"></span><span class="c dim">-H "Authorization: Bearer $KDEPS_API_AUTH_TOKEN" \</span></div>
          <div class="tl"><span class="pad"></span><span class="c dim">-d '{"q": "What is entropy, in one sentence?"}'</span></div>
          <div class="tl">&nbsp;</div>
          <div class="tl resp">{"success": true, "data": {"answer": "Entropy measures how many microscopic arrangements are consistent with a system's macroscopic state."}}</div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.hero-code-section {
  padding: 0 24px 64px;
}

.hero-code-container {
  max-width: 960px;
  margin: 0 auto;
}

.hero-window {
  font-family: var(--vp-font-family-mono);
  font-size: 12px;
  line-height: 1.55;
  background: #070707;
  border: 1px solid rgba(255,255,255,0.08);
  border-radius: 2px;
  overflow: hidden;
  width: 100%;
  box-shadow: var(--vp-shadow-3);
}

.titlebar {
  display: flex;
  align-items: stretch;
  gap: 10px;
  padding: 6px 14px 0;
  background: rgba(0,229,255,0.04);
  border-bottom: 1px solid rgba(255,255,255,0.06);
}

.dots {
  display: flex;
  align-items: center;
  gap: 5px;
  padding-bottom: 6px;
  flex-shrink: 0;
}

.dots span { width: 8px; height: 8px; border-radius: 50%; }
.dots .r { background: #FF5F57; }
.dots .y { background: #FFBD2E; }
.dots .g { background: #28CA42; }

.tabs { display: flex; }

.tab {
  font-family: var(--vp-font-family-mono);
  font-size: 10px;
  padding: 4px 10px 6px;
  color: rgba(255,255,255,0.3);
  background: transparent;
  border: none;
  border-bottom: 2px solid transparent;
  cursor: pointer;
  white-space: nowrap;
  transition: color 0.15s;
}

.tab:hover { color: rgba(255,255,255,0.6); }
.tab:focus-visible { outline: 1px solid rgba(0, 229, 255, 0.5); outline-offset: -2px; }
.tab.active { color: rgba(255,255,255,0.85); border-bottom-color: var(--vp-c-brand-1); }

.code-body {
  margin: 0;
  padding: 12px 16px;
  color: #c8cce8;
  overflow-x: auto;
  white-space: pre;
  height: 260px;
  border-bottom: 1px solid rgba(255,255,255,0.06);
}

.code-body code {
  font-family: inherit;
  font-size: inherit;
  color: inherit;
  background: transparent;
  padding: 0;
  border: none;
}

:deep(.k) { color: #FF2D78; }
:deep(.s) { color: #FFD60A; }
:deep(.n) { color: #FF9500; }
:deep(.o) { color: var(--vp-c-brand-1); }
:deep(.t) { color: var(--vp-c-brand-1); }
:deep(.d) { color: rgba(200, 204, 232, 0.4); }
:deep(.pr) { color: var(--vp-c-brand-1); user-select: none; }
:deep(.re) { color: rgba(200, 204, 232, 0.75); }

.terminal { padding: 10px 16px; background: rgba(0,0,0,0.25); }

.tl {
  display: flex;
  gap: 8px;
  line-height: 1.5;
  white-space: pre;
}

.p { color: var(--vp-c-brand-1); user-select: none; flex-shrink: 0; }
.c { color: #c8cce8; }
.c.dim { color: rgba(200,204,232,0.5); }
.tl.dim { color: rgba(200,204,232,0.35); padding-left: 16px; }
.pad { display: inline-block; width: 16px; flex-shrink: 0; }
.resp { color: #FFD60A; white-space: pre-wrap; word-break: break-word; }

@media (max-width: 960px) {
  .hero-code-section { padding: 0 16px 48px; }
}
@media (max-width: 640px) {
  .hero-code-section { padding: 0 12px 40px; }
  .hero-window { font-size: 11px; }
  .tab { font-size: 9px; padding: 4px 7px 6px; }
}
</style>
