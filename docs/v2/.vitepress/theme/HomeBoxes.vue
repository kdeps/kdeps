<!--
  Copyright 2026 Kdeps, KvK 94834768
  Licensed under the Apache License, Version 2.0
-->
<script setup>
import { onMounted, onUnmounted, ref } from 'vue'

// Scenes run only while the grid is on screen and motion is allowed;
// otherwise each scene shows its finished state.
const root = ref(null)
const play = ref(false)
let io

onMounted(() => {
  if (!document.documentElement.classList.contains('kd-motion')) return
  io = new IntersectionObserver(([e]) => { play.value = e.isIntersecting }, { threshold: 0.15 })
  io.observe(root.value)
})

onUnmounted(() => io?.disconnect())
</script>

<template>
  <section class="boxes">
    <div class="container">
      <p class="section-eyebrow">inside kdeps</p>
      <h2 class="section-title">What one YAML file gives you</h2>
      <p class="section-sub">Each box is something you get without writing glue code.</p>

      <div ref="root" :class="['grid', { play }]">
        <div class="box">
          <div class="scene modes" aria-hidden="true">
            <span class="chip strong">workflow.yaml</span>
            <div class="tracks"><i class="track"><b></b></i><i class="track"><b></b></i></div>
            <div class="row">
              <span class="chip">kdeps run<em>API</em></span>
              <span class="chip">kdeps .<em>agent REPL</em></span>
            </div>
          </div>
          <h3>Two modes, one file</h3>
          <p><code>kdeps run</code> serves it as an API. <code>kdeps .</code> loads it as a tool in the agent loop.</p>
        </div>

        <div class="box">
          <div class="scene tools" aria-hidden="true">
            <div class="line prompt">&gt; fix the failing test</div>
            <div class="line step s1"><span class="tool">read_file</span> handler_test.go<span class="ok">✓</span></div>
            <div class="line step s2"><span class="tool">edit_file</span> line 41<span class="ok">✓</span></div>
            <div class="line step s3"><span class="tool">bash_exec</span> go test<span class="ok">PASS</span></div>
          </div>
          <h3>An agent with tools</h3>
          <p>The agent loop reads, edits and runs commands through fenced kdeps tools.</p>
        </div>

        <div class="box">
          <div class="scene models" aria-hidden="true">
            <div class="line prompt">&gt; /model</div>
            <div class="swap">
              <span class="chip strong m1">llama3.2 - local</span>
              <span class="chip strong m2">claude-sonnet - Anthropic</span>
              <span class="chip strong m3">free model - OpenRouter</span>
            </div>
          </div>
          <h3>Any model, any time</h3>
          <p>Switch between local and cloud models mid-session with <code>/model</code>.</p>
        </div>

        <div class="box">
          <div class="scene judges" aria-hidden="true">
            <div class="row">
              <span class="chip">answer</span>
              <span class="arrow">→</span>
              <div class="stack">
                <span class="chip badge j1">correctness ✓</span>
                <span class="chip badge j2">completeness ✓</span>
                <span class="chip badge j3">security ✓</span>
              </div>
            </div>
            <span class="chip strong approved">approved</span>
          </div>
          <h3>Judges check the answer</h3>
          <p>A judge panel reviews the final answer and sends it back for a revision on a reject.</p>
        </div>

        <div class="box">
          <div class="scene offline" aria-hidden="true">
            <div class="row">
              <span class="led"></span>
              <span class="chip">llamafile - llama3.2</span>
            </div>
            <div class="bars"><i></i><i></i><i></i><i></i><i></i><i></i><i></i></div>
            <span class="net">network: off</span>
          </div>
          <h3>Runs offline</h3>
          <p>llamafile runs the model on your machine. No API key, no data leaves it.</p>
        </div>

        <div class="box">
          <div class="scene ship" aria-hidden="true">
            <span class="chip strong">workflow.yaml</span>
            <div class="row">
              <span class="chip t1">Docker</span>
              <span class="chip t2">Kubernetes</span>
              <span class="chip t3">ISO</span>
              <span class="chip t4">binary</span>
            </div>
          </div>
          <h3>Ship it anywhere</h3>
          <p>Export the same workflow as a Docker image, Kubernetes manifests, an ISO or one binary.</p>
        </div>
      </div>
    </div>
  </section>
</template>

<style scoped>
.boxes {
  padding: 72px 24px;
  border-top: 1px solid rgba(255, 255, 255, 0.06);
}

.container { max-width: 960px; margin: 0 auto; }

.section-eyebrow {
  font-family: var(--vp-font-family-mono);
  font-size: 11px;
  font-weight: 600;
  letter-spacing: 0.12em;
  text-transform: uppercase;
  color: var(--vp-c-brand-1);
  margin: 0 0 10px;
}

.section-title {
  font-size: 36px;
  font-weight: 700;
  letter-spacing: -0.02em;
  color: var(--vp-c-text-1);
  margin: 0 0 8px;
}

.section-sub {
  font-size: 16px;
  color: var(--vp-c-text-2);
  margin: 0 0 40px;
}

.grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 16px;
}

.box {
  background: var(--vp-c-bg-soft);
  border: 1px solid rgba(255, 255, 255, 0.07);
  border-radius: 2px;
  padding: 16px;
  transition: transform 0.15s ease, border-color 0.15s ease;
}

.box:hover {
  transform: translateY(-2px);
  border-color: rgba(0, 229, 255, 0.3);
}

.box h3 {
  font-size: 17px;
  font-weight: 600;
  color: var(--vp-c-text-1);
  margin: 16px 0 6px;
}

.box p {
  font-size: 14px;
  line-height: 1.6;
  color: var(--vp-c-text-2);
  margin: 0;
}

.box p code {
  font-family: var(--vp-font-family-mono);
  font-size: 12px;
  background: rgba(255, 255, 255, 0.05);
  padding: 1px 5px;
  border-radius: 2px;
  border: 1px solid rgba(255, 255, 255, 0.08);
  color: var(--vp-c-brand-1);
}

/* ---- scene frame and shared pieces ---- */
.scene {
  height: 150px;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 10px;
  background: rgba(0, 0, 0, 0.2);
  border: 1px solid rgba(255, 255, 255, 0.06);
  border-radius: 2px;
  font-family: var(--vp-font-family-mono);
  font-size: 11px;
  color: var(--vp-c-text-2);
  overflow: hidden;
}

.row { display: flex; gap: 6px; align-items: center; flex-wrap: wrap; justify-content: center; }

.chip {
  padding: 3px 8px;
  border: 1px solid rgba(255, 255, 255, 0.1);
  border-radius: 2px;
  white-space: nowrap;
}
.chip.strong { color: var(--vp-c-brand-1); border-color: rgba(0, 229, 255, 0.35); }
.chip em { font-style: normal; color: var(--vp-c-text-3); margin-left: 6px; }

.line { width: 82%; white-space: nowrap; }
.prompt { color: var(--vp-c-brand-1); }

/* Scene animations are declared under .grid.play, so they run only on screen. */

/* ---- two modes: a dot runs down each branch ---- */
.tracks { display: flex; gap: 64px; }
.track {
  position: relative;
  width: 1px;
  height: 22px;
  background: rgba(255, 255, 255, 0.12);
}
.track b {
  position: absolute;
  left: -2px;
  top: 0;
  width: 5px;
  height: 5px;
  border-radius: 50%;
  background: var(--vp-c-brand-1);
  opacity: 0;
}
.grid.play .track b { animation: kd-drop 1.6s ease-in infinite; }
.grid.play .track + .track b { animation-delay: 0.8s; }
@keyframes kd-drop {
  0% { opacity: 0; transform: translateY(0); }
  20% { opacity: 1; }
  100% { opacity: 0; transform: translateY(18px); }
}

/* ---- tools: calls appear one after another ---- */
.tools { align-items: stretch; padding-left: 9%; gap: 7px; }
.tool { color: var(--vp-c-brand-1); margin-right: 6px; }
.ok { color: var(--vp-c-brand-1); margin-left: 6px; }
.grid.play .step { animation: kd-cycle-in 6s ease infinite both; }
.grid.play .s2 { animation-delay: 0.9s; }
.grid.play .s3 { animation-delay: 1.8s; }
@keyframes kd-cycle-in {
  0% { opacity: 0; transform: translateY(4px); }
  8%, 80% { opacity: 1; transform: none; }
  92%, 100% { opacity: 0; transform: none; }
}

/* ---- models: one chip swaps for the next ---- */
.swap { position: relative; height: 24px; width: 100%; }
.swap .chip {
  position: absolute;
  left: 50%;
  translate: -50% 0;
}
.swap .m2, .swap .m3 { visibility: hidden; }
.grid.play .swap .chip { visibility: visible; opacity: 0; animation: kd-swap 6s ease infinite; }
.grid.play .swap .m2 { animation-delay: 2s; }
.grid.play .swap .m3 { animation-delay: 4s; }
@keyframes kd-swap {
  0% { opacity: 0; transform: translateY(8px); }
  5%, 28% { opacity: 1; transform: none; }
  33%, 100% { opacity: 0; transform: translateY(-8px); }
}

/* ---- judges: verdicts pop in, then approval ---- */
.judges .row { flex-wrap: nowrap; gap: 10px; }
.stack { display: flex; flex-direction: column; gap: 4px; align-items: flex-start; }
.arrow { color: var(--vp-c-text-3); }
.badge { color: var(--vp-c-text-1); }
.grid.play .badge,
.grid.play .approved { animation: kd-pop-cycle 6s cubic-bezier(0.2, 1.4, 0.4, 1) infinite both; }
.grid.play .j2 { animation-delay: 0.5s; }
.grid.play .j3 { animation-delay: 1s; }
.grid.play .approved { animation-delay: 1.7s; }
@keyframes kd-pop-cycle {
  0% { opacity: 0; transform: scale(0.8); }
  6%, 82% { opacity: 1; transform: none; }
  92%, 100% { opacity: 0; transform: none; }
}

/* ---- offline: a status light and a generating meter ---- */
.led {
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background: var(--vp-c-brand-1);
}
.grid.play .led { animation: kd-blink 1.2s step-end infinite; }
@keyframes kd-blink { 50% { opacity: 0.25; } }
.bars { display: flex; gap: 4px; align-items: flex-end; height: 26px; }
.bars i {
  width: 5px;
  height: 60%;
  background: rgba(0, 229, 255, 0.55);
  border-radius: 1px;
}
.grid.play .bars i { animation: kd-bar 0.7s ease-in-out infinite alternate; }
.bars i:nth-child(2) { animation-delay: -0.2s; }
.bars i:nth-child(3) { animation-delay: -0.5s; }
.bars i:nth-child(4) { animation-delay: -0.1s; }
.bars i:nth-child(5) { animation-delay: -0.4s; }
.bars i:nth-child(6) { animation-delay: -0.3s; }
.bars i:nth-child(7) { animation-delay: -0.6s; }
@keyframes kd-bar {
  from { height: 20%; }
  to { height: 100%; }
}
.net { color: var(--vp-c-text-3); text-decoration: line-through; }

/* ---- ship: each target lights up in turn ---- */
.grid.play .ship .row .chip { animation: kd-lit 4s ease infinite; }
.grid.play .ship .t2 { animation-delay: 1s; }
.grid.play .ship .t3 { animation-delay: 2s; }
.grid.play .ship .t4 { animation-delay: 3s; }
@keyframes kd-lit {
  0%, 28%, 100% { color: var(--vp-c-text-2); border-color: rgba(255, 255, 255, 0.1); }
  5%, 22% { color: var(--vp-c-brand-1); border-color: var(--vp-c-brand-1); }
}

@media (max-width: 960px) {
  .grid { grid-template-columns: repeat(2, 1fr); }
}
@media (max-width: 640px) {
  .boxes { padding: 48px 16px; }
  .grid { grid-template-columns: 1fr; }
}
</style>
