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
    name: 'local agent',
    html: [
      `${P('$')} kdeps`,
      ``,
      `${D('kdeps v2.x.x  |  Local agent mode')}`,
      `${D('Model: llama3.2 (Ollama)  |  Type /help for commands')}`,
      ``,
      `${T('>')} write a Go function that parses a CSV file`,
      ``,
      `${R('Sure. Here\'s an idiomatic Go CSV parser...')}`,
      ``,
      `${R('func ParseCSV(r io.Reader) ([][]string, error) {')}`,
      `${R('    reader := csv.NewReader(r)')}`,
      `${R('    return reader.ReadAll()')}`,
      `${R('}')}`,
      ``,
      `${T('>')} /model claude-opus-4-8`,
      `${D('Switched to claude-opus-4-8 (Anthropic)')}`,
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
      `  ${K('name')}${O(': ')}${S('summarizer')}`,
      `  ${K('version')}${O(': ')}${S('"1.0.0"')}`,
      `  ${K('targetActionId')}${O(': ')}${S('summarize')}`,
      ``,
      `${K('settings')}${O(':')}`,
      `  ${K('apiServer')}${O(':')}`,
      `    ${K('portNum')}${O(': ')}${N('16395')}`,
      `    ${K('routes')}${O(':')}`,
      `      ${O('- ')}${K('path')}${O(': ')}${S('/summarize')}`,
      `        ${K('methods')}${O(': ')}${S('[POST]')}`,
    ].join('\n'),
  },
  {
    name: 'resources/fetch.yaml',
    html: [
      `${K('actionId')}${O(': ')}${S('fetch')}`,
      `${K('httpClient')}${O(':')}`,
      `  ${K('method')}${O(': ')}${S('GET')}`,
      `  ${K('url')}${O(': ')}${S('"{{ get(\'url\') }}"')}`,
      `  ${K('timeout')}${O(': ')}${S('10s')}`,
      '', '', '', '', '', '', '', '',
    ].join('\n'),
  },
  {
    name: 'resources/summarize.yaml',
    html: [
      `${K('actionId')}${O(': ')}${S('summarize')}`,
      `${K('requires')}${O(': ')}${S('[fetch]')}`,
      `${K('chat')}${O(':')}`,
      `  ${K('model')}${O(': ')}${S('llama3.2:1b')}`,
      `  ${K('prompt')}${O(': ')}${S('"Summarize: {{ output(\'fetch\').body }}"')}`,
      `${K('apiResponse')}${O(':')}`,
      `  ${K('response')}${O(': ')}${S('"{{ output(\'summarize\') }}"')}`,
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
          <div class="tl"><span class="p">$</span><span class="c">kdeps run workflow.yaml</span></div>
          <div class="tl dim">Listening on :16395</div>
          <div class="tl">&nbsp;</div>
          <div class="tl"><span class="p">$</span><span class="c">curl -s -X POST localhost:16395/summarize \</span></div>
          <div class="tl"><span class="pad"></span><span class="c dim">-H "Authorization: Bearer $KDEPS_API_AUTH_TOKEN" \</span></div>
          <div class="tl"><span class="pad"></span><span class="c dim">-H "Content-Type: application/json" \</span></div>
          <div class="tl"><span class="pad"></span><span class="c dim">-d '{"url": "https://example.com"}'</span></div>
          <div class="tl">&nbsp;</div>
          <div class="tl resp">{"success": true, "data": {"response": "Example.com is used for illustrative examples in documentation."}}</div>
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
