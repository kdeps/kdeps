<!--
  Copyright 2026 Kdeps, KvK 94834768
  Licensed under the Apache License, Version 2.0
-->
<script setup>
import { ref } from 'vue'

const active = ref(0)

const files = [
  {
    name: 'workflow.yaml',
    lines: [
      { t: 'key', v: 'apiVersion' }, { t: 'op', v: ': ' }, { t: 'str', v: 'kdeps.io/v1' },
      { t: 'key', v: 'kind' },       { t: 'op', v: ': ' }, { t: 'str', v: 'Workflow' },
      { t: 'break' },
      { t: 'key', v: 'metadata' }, { t: 'op', v: ':' },
      { t: 'indent', v: '  ' }, { t: 'key', v: 'name' },           { t: 'op', v: ': ' }, { t: 'str', v: 'summarizer' },
      { t: 'indent', v: '  ' }, { t: 'key', v: 'version' },        { t: 'op', v: ': ' }, { t: 'str', v: '"1.0.0"' },
      { t: 'indent', v: '  ' }, { t: 'key', v: 'targetActionId' }, { t: 'op', v: ': ' }, { t: 'str', v: 'summarize' },
      { t: 'break' },
      { t: 'key', v: 'settings' }, { t: 'op', v: ':' },
      { t: 'indent', v: '  ' }, { t: 'key', v: 'apiServer' }, { t: 'op', v: ':' },
      { t: 'indent', v: '    ' }, { t: 'key', v: 'portNum' }, { t: 'op', v: ': ' }, { t: 'num', v: '16395' },
      { t: 'indent', v: '    ' }, { t: 'key', v: 'routes' }, { t: 'op', v: ':' },
      { t: 'indent', v: '      ' }, { t: 'op', v: '- ' }, { t: 'key', v: 'path' },    { t: 'op', v: ': ' }, { t: 'str', v: '/summarize' },
      { t: 'indent', v: '        ' }, { t: 'key', v: 'methods' }, { t: 'op', v: ': ' }, { t: 'str', v: '[POST]' },
    ]
  },
  {
    name: 'resources/fetch.yaml',
    lines: [
      { t: 'key', v: 'actionId' }, { t: 'op', v: ': ' }, { t: 'str', v: 'fetch' },
      { t: 'key', v: 'httpClient' }, { t: 'op', v: ':' },
      { t: 'indent', v: '  ' }, { t: 'key', v: 'method' },  { t: 'op', v: ': ' }, { t: 'str', v: 'GET' },
      { t: 'indent', v: '  ' }, { t: 'key', v: 'url' },     { t: 'op', v: ': ' }, { t: 'str', v: '"{{ get(\'url\') }}"' },
      { t: 'indent', v: '  ' }, { t: 'key', v: 'timeout' }, { t: 'op', v: ': ' }, { t: 'str', v: '10s' },
    ]
  },
  {
    name: 'resources/summarize.yaml',
    lines: [
      { t: 'key', v: 'actionId' }, { t: 'op', v: ': ' }, { t: 'str', v: 'summarize' },
      { t: 'key', v: 'requires' }, { t: 'op', v: ': ' }, { t: 'str', v: '[fetch]' },
      { t: 'key', v: 'chat' }, { t: 'op', v: ':' },
      { t: 'indent', v: '  ' }, { t: 'key', v: 'model' },  { t: 'op', v: ': ' }, { t: 'str', v: 'llama3.2:1b' },
      { t: 'indent', v: '  ' }, { t: 'key', v: 'prompt' }, { t: 'op', v: ': ' }, { t: 'str', v: '"Summarize: {{ output(\'fetch\').body }}"' },
      { t: 'key', v: 'apiResponse' }, { t: 'op', v: ':' },
      { t: 'indent', v: '  ' }, { t: 'key', v: 'response' }, { t: 'op', v: ': ' }, { t: 'str', v: '"{{ output(\'summarize\') }}"' },
    ]
  },
]

// Group flat tokens into display lines
function toDisplayLines(tokens) {
  const result = []
  let current = []
  for (const tok of tokens) {
    if (tok.t === 'break') {
      result.push(current)
      current = []
      result.push([{ t: 'blank' }])
    } else if (tok.t === 'indent') {
      result.push(current)
      current = [tok]
    } else {
      current.push(tok)
    }
  }
  if (current.length) result.push(current)
  return result.filter(l => l.length > 0)
}
</script>

<template>
  <div class="hero-window">
    <!-- File tabs -->
    <div class="titlebar">
      <div class="dots">
        <span class="r"></span>
        <span class="y"></span>
        <span class="g"></span>
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

    <!-- Code body -->
    <pre class="code-body"><code>
<template v-for="(line, li) in toDisplayLines(files[active].lines)" :key="li">
<span v-if="line[0]?.t === 'blank'">&nbsp;</span>
<span v-else><template v-for="(tok, ti) in line" :key="ti"><span v-if="tok.t === 'indent'">{{ tok.v }}</span><span v-else :class="tok.t">{{ tok.v }}</span></template></span>
</template>
</code></pre>

    <!-- Terminal section -->
    <div class="terminal">
      <div class="term-line">
        <span class="prompt">$</span>
        <span class="cmd">kdeps run workflow.yaml</span>
      </div>
      <div class="term-line out">Listening on :16395</div>
      <div class="term-spacer"></div>
      <div class="term-line">
        <span class="prompt">$</span>
        <span class="cmd">curl -s -X POST localhost:16395/summarize \</span>
      </div>
      <div class="term-line">
        <span class="term-pad"></span>
        <span class="cmd dim">-H "Content-Type: application/json" \</span>
      </div>
      <div class="term-line">
        <span class="term-pad"></span>
        <span class="cmd dim">-d '{"url": "https://example.com"}'</span>
      </div>
      <div class="term-spacer"></div>
      <div class="term-line resp">{</div>
      <div class="term-line resp">&nbsp;&nbsp;"success": true,</div>
      <div class="term-line resp">&nbsp;&nbsp;"data": { "response": "Example.com is a domain used for illustrative examples in documentation." }</div>
      <div class="term-line resp">}</div>
    </div>
  </div>
</template>

<style scoped>
.hero-window {
  font-family: var(--vp-font-family-mono);
  font-size: 12px;
  line-height: 1.6;
  background: #070707;
  border: 1px solid rgba(255, 255, 255, 0.08);
  border-left: 2px solid var(--vp-c-brand-1);
  border-radius: 2px;
  overflow: hidden;
  width: 100%;
  max-width: 520px;
  box-shadow: var(--vp-shadow-3);
}

/* Title bar */
.titlebar {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 6px 14px 0;
  background: rgba(0, 229, 255, 0.04);
  border-bottom: 1px solid rgba(255, 255, 255, 0.06);
}

.dots {
  display: flex;
  gap: 5px;
  padding-bottom: 6px;
  flex-shrink: 0;
}

.dots span {
  width: 8px;
  height: 8px;
  border-radius: 50%;
}

.dots .r { background: #FF5F57; }
.dots .y { background: #FFBD2E; }
.dots .g { background: #28CA42; }

.tabs {
  display: flex;
  gap: 0;
  overflow-x: auto;
}

.tab {
  font-family: var(--vp-font-family-mono);
  font-size: 10px;
  padding: 4px 10px 6px;
  color: rgba(255, 255, 255, 0.3);
  background: transparent;
  border: none;
  border-bottom: 2px solid transparent;
  cursor: pointer;
  white-space: nowrap;
  transition: color 0.15s;
}

.tab:hover {
  color: rgba(255, 255, 255, 0.6);
}

.tab.active {
  color: rgba(255, 255, 255, 0.85);
  border-bottom-color: var(--vp-c-brand-1);
}

/* Code body */
.code-body {
  margin: 0;
  padding: 12px 16px;
  color: #c8cce8;
  overflow-x: auto;
  white-space: pre;
  min-height: 120px;
  border-bottom: 1px solid rgba(255, 255, 255, 0.06);
}

.code-body code {
  font-family: inherit;
  font-size: inherit;
  color: inherit;
  background: transparent;
  padding: 0;
  border: none;
}

.key  { color: #FF2D78; }
.str  { color: #FFD60A; }
.num  { color: #FF9500; }
.op   { color: var(--vp-c-brand-1); }

/* Terminal section */
.terminal {
  padding: 10px 16px;
  background: rgba(0, 0, 0, 0.3);
}

.term-line {
  display: flex;
  gap: 8px;
  white-space: pre;
  line-height: 1.5;
}

.term-spacer {
  height: 6px;
}

.prompt {
  color: var(--vp-c-brand-1);
  user-select: none;
  flex-shrink: 0;
}

.cmd {
  color: #c8cce8;
}

.cmd.dim {
  color: rgba(200, 204, 232, 0.5);
}

.term-pad {
  display: inline-block;
  width: 16px;
  flex-shrink: 0;
}

.out {
  color: rgba(200, 204, 232, 0.35);
  padding-left: 16px;
}

.resp {
  color: #FFD60A;
  padding-left: 0;
  white-space: pre-wrap;
  word-break: break-word;
}

@media (max-width: 960px) {
  .hero-window { max-width: 100%; }
}

@media (max-width: 640px) {
  .hero-window { font-size: 11px; }
  .tab { font-size: 9px; padding: 4px 7px 6px; }
}
</style>
