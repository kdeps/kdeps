<!--
  Copyright 2026 Kdeps, KvK 94834768
  Licensed under the Apache License, Version 2.0
-->
<script setup>
import { computed, onMounted, onUnmounted, ref } from 'vue'

const active = ref(0)
const root = ref(null)
// Terminal lines shown so far. Starts at 0 for SSR; CSS hides unshown lines
// only under html.kd-motion, so static visitors see the whole transcript.
const shown = ref(0)
let touched = false
let paused = false
let tabTimer
let termTimer
let io

const lines = computed(() => files[active.value].html.split('\n'))

function pick(i) {
  touched = true
  active.value = i
}

function playTerminal() {
  if (termTimer) return
  termTimer = setInterval(() => {
    shown.value++
    if (shown.value >= term.length) clearInterval(termTimer)
  }, 420)
}

onMounted(() => {
  if (!document.documentElement.classList.contains('kd-motion')) {
    shown.value = term.length
    return
  }
  io = new IntersectionObserver(([e]) => {
    paused = !e.isIntersecting
    if (e.isIntersecting) playTerminal()
  }, { threshold: 0.3 })
  io.observe(root.value)
  // Cycle the file tabs like a demo until the visitor picks one.
  tabTimer = setInterval(() => {
    if (!touched && !paused) active.value = (active.value + 1) % files.length
  }, 7000)
})

onUnmounted(() => {
  clearInterval(tabTimer)
  clearInterval(termTimer)
  io?.disconnect()
})

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
    name: 'resources/llm.yaml',
    html: [
      `${K('actionId')}${O(': ')}${S('llm')}`,
      `${K('name')}${O(': ')}${S('Ask the AI')}`,
      `${K('validations')}${O(':')}`,
      `  ${K('check')}${O(':')}`,
      `    ${O('- ')}${S("get('q') != ''")}   ${D('# 400 before any model call')}`,
      `  ${K('error')}${O(':')}`,
      `    ${K('code')}${O(': ')}${N('400')}`,
      `    ${K('message')}${O(': ')}${S('"\'q\' is required"')}`,
      `${K('chat')}${O(':')}`,
      `  ${K('model')}${O(':')}`,
      `    ${K('name')}${O(': ')}${S('ministral3')}   ${D('# local, no API key')}`,
      `  ${K('context')}${O(':')}`,
      `    ${K('prompt')}${O(': ')}${S('"{{ get(\'q\') }}"')}`,
      `  ${K('loop')}${O(':')}   ${D('# a bounded agent step')}`,
      `    ${K('tools')}${O(': ')}${S('[read_file]')}`,
      `    ${K('maxRounds')}${O(': ')}${N('5')}`,
    ].join('\n'),
  },
  {
    name: 'evals',
    html: [
      `${K('evals')}${O(':')}   ${D('# kdeps test runs these in the process')}`,
      `  ${O('- ')}${K('name')}${O(': ')}${S('needs-q')}`,
      `    ${K('request')}${O(':')}`,
      `      ${K('method')}${O(': ')}${S('POST')}`,
      `      ${K('path')}${O(': ')}${S('/api/v1/chat')}`,
      `      ${K('body')}${O(': ')}${S('{}')}`,
      `    ${K('assert')}${O(':')}`,
      `      ${K('status')}${O(': ')}${N('400')}`,
      '', '', '', '', '', '', '',
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
      `  ${K('targetActionId')}${O(': ')}${S('response')}   ${D('# its output is the HTTP response')}`,
      ``,
      `${K('deploy')}${O(':')}`,
      `  ${K('apiServer')}${O(':')}`,
      `    ${K('portNum')}${O(': ')}${N('16395')}`,
      `    ${K('routes')}${O(':')}`,
      `      ${O('- ')}${K('path')}${O(': ')}${S('/api/v1/chat')}`,
      `        ${K('methods')}${O(': ')}${S('[POST]')}`,
    ].join('\n'),
  },
  {
    name: 'ship',
    html: [
      `${P('$')} kdeps bundle build .`,
      `${D('# container image, same workflow.yaml')}`,
      ``,
      `${P('$')} kdeps bundle package .`,
      `${D('# writes chat-api-1.0.0.kdeps')}`,
      ``,
      `${P('$')} kdeps bundle prepackage chat-api-1.0.0.kdeps --include-models`,
      `${D('# the llamafile is inside the binary')}`,
      ``,
      `${P('$')} KDEPS_API_AUTH_TOKEN=... kdeps export iso .`,
      `${D('# bootable ISO with the token inside; boots offline')}`,
      '', '', '', '',
    ].join('\n'),
  },
]

const term = [
  { cls: '', html: '<span class="p">$</span><span class="c">export KDEPS_API_AUTH_TOKEN=dev-token</span>' },
  { cls: '', html: '<span class="p">$</span><span class="c">kdeps test .</span>' },
  { cls: 'dim', html: 'Running 1 eval(s) for chat-api' },
  { cls: '', html: '&nbsp;' },
  { cls: 'dim', html: '  PASS  needs-q (400, 1ms)' },
  { cls: '', html: '&nbsp;' },
  { cls: 'dim', html: '1 passed, 0 failed' },
  { cls: '', html: '<span class="p">$</span><span class="c">kdeps run .</span>' },
  { cls: 'dim', html: 'Starting HTTP server on 0.0.0.0:16395' },
  { cls: '', html: '<span class="p">$</span>' },
]

const providers = [
  'Ollama', 'llamafile', 'OpenAI', 'Anthropic', 'Google', 'OpenRouter', 'Groq',
  'Mistral', 'DeepSeek', 'xAI', 'Cohere', 'Together', 'Perplexity',
  'Hugging Face', 'Cloudflare', 'Bedrock', 'watsonx',
]
</script>

<template>
  <div class="hero-code-section">
    <div ref="root" class="hero-code-container">
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
              @click="pick(i)"
            >{{ f.name }}</button>
          </div>
        </div>

        <pre class="code-body"><code :key="active"><span
          v-for="(l, i) in lines"
          :key="i"
          class="ln"
          :style="{ animationDelay: `${i * 25}ms` }"
          v-html="l || ' '"
        ></span></code></pre>

        <div class="terminal">
          <div
            v-for="(t, i) in term"
            :key="i"
            :class="['tl', t.cls, { on: i < shown }]"
          ><span v-html="t.html"></span><span v-if="i === shown - 1" class="caret"></span></div>
        </div>
      </div>

      <div class="providers">
        <p class="providers-label">runs on</p>
        <div class="marquee">
          <div class="track">
            <span v-for="p in providers" :key="p">{{ p }}</span>
            <span v-for="p in providers" :key="'dup-' + p" class="dup" aria-hidden="true">{{ p }}</span>
          </div>
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
  height: 300px;
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

/* Terminal lines are v-html, so their inner spans need :deep. */
.tl > span:first-child { display: contents; }
:deep(.p) { color: var(--vp-c-brand-1); user-select: none; flex-shrink: 0; }
:deep(.c) { color: #c8cce8; }
:deep(.c.dim) { color: rgba(200,204,232,0.5); }
.tl.dim { color: rgba(200,204,232,0.35); padding-left: 16px; }
:deep(.pad) { display: inline-block; width: 16px; flex-shrink: 0; }
.resp { color: #FFD60A; white-space: pre-wrap; word-break: break-word; }

/* Code lines slide in on each tab change. */
.ln { display: block; }
html.kd-motion .ln { animation: kd-line 0.3s ease both; }
@keyframes kd-line {
  from { opacity: 0; transform: translateX(-6px); }
  to { opacity: 1; transform: none; }
}

/* Terminal plays line by line once in view. */
html.kd-motion .tl { transition: opacity 0.25s ease; }
html.kd-motion .tl:not(.on) { opacity: 0; }
.caret {
  display: none;
  width: 7px;
  height: 1.1em;
  margin-left: 2px;
  background: var(--vp-c-brand-1);
  vertical-align: text-bottom;
}
html.kd-motion .caret {
  display: inline-block;
  animation: kd-caret 1s step-end infinite;
}
@keyframes kd-caret { 50% { opacity: 0; } }

/* Provider strip: a wrapped list when static, a looping marquee with motion. */
.providers { margin-top: 28px; }
.providers-label {
  font-family: var(--vp-font-family-mono);
  font-size: 11px;
  font-weight: 600;
  letter-spacing: 0.12em;
  text-transform: uppercase;
  color: var(--vp-c-text-3);
  text-align: center;
  margin: 0 0 12px;
}
.marquee { overflow: hidden; }
.track {
  display: flex;
  flex-wrap: wrap;
  justify-content: center;
  gap: 8px 36px;
  font-family: var(--vp-font-family-mono);
  font-size: 13px;
  color: var(--vp-c-text-2);
  white-space: nowrap;
}
.dup { display: none; }
html.kd-motion .marquee {
  mask-image: linear-gradient(90deg, transparent, #000 12%, #000 88%, transparent);
}
html.kd-motion .track {
  flex-wrap: nowrap;
  justify-content: flex-start;
  width: max-content;
  animation: kd-marquee 40s linear infinite;
}
html.kd-motion .dup { display: inline; }
html.kd-motion .marquee:hover .track { animation-play-state: paused; }
/* Each item carries its own trailing gap so -50% lands exactly on the copy. */
html.kd-motion .track { gap: 0; }
html.kd-motion .track span { padding-right: 36px; }
@keyframes kd-marquee { to { transform: translateX(-50%); } }

@media (max-width: 960px) {
  .hero-code-section { padding: 0 16px 48px; }
}
@media (max-width: 640px) {
  .hero-code-section { padding: 0 12px 40px; }
  .hero-window { font-size: 11px; }
  .tab { font-size: 9px; padding: 4px 7px 6px; }
}
</style>
