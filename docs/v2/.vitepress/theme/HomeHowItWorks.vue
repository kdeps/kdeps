<!--
  Copyright 2026 Kdeps, KvK 94834768
  Licensed under the Apache License, Version 2.0
-->
<script setup>
import { onMounted, onUnmounted, ref } from 'vue'

// Index of the lit step; -1 keeps all steps in their static style.
const lit = ref(-1)
const root = ref(null)
let timer
let io

onMounted(() => {
  if (!document.documentElement.classList.contains('kd-motion')) return
  io = new IntersectionObserver(([e]) => {
    clearInterval(timer)
    if (!e.isIntersecting) return
    lit.value = 0
    timer = setInterval(() => { lit.value = (lit.value + 1) % 3 }, 1600)
  }, { threshold: 0.4 })
  io.observe(root.value)
})

onUnmounted(() => {
  clearInterval(timer)
  io?.disconnect()
})
</script>

<template>
  <section class="how-it-works">
    <div class="container">
      <p class="section-eyebrow">getting started</p>
      <h2 class="section-title">How it works</h2>
      <p class="section-sub">Write YAML. Run it. Ship the same file.</p>

      <div ref="root" class="steps">
        <div :class="['step', { on: lit === 0 }]">
          <div class="step-num">01</div>
          <div class="step-body">
            <h3>Write YAML</h3>
            <p>Declare resources: chat, HTTP, Python, SQL. Connect them with <code>requires:</code> under <code>flow:</code>. A <code>chat</code> with <code>loop:</code> is an agent.</p>
          </div>
        </div>

        <div :class="['step-arrow', { on: lit === 1 }]">→</div>

        <div :class="['step', { on: lit === 1 }]">
          <div class="step-num">02</div>
          <div class="step-body">
            <h3>Run it</h3>
            <p><code>kdeps</code> runs <code>./workflow.yaml</code>: as an API, a bot, a file processor or an interactive prompt.</p>
          </div>
        </div>

        <div :class="['step-arrow', { on: lit === 2 }]">→</div>

        <div :class="['step', { on: lit === 2 }]">
          <div class="step-num">03</div>
          <div class="step-body">
            <h3>Ship it</h3>
            <p>Export it as Docker, Kubernetes, ISO, or a binary. All use the same workflow.yaml. You do not rewrite or reconfigure it.</p>
          </div>
        </div>
      </div>
    </div>
  </section>
</template>

<style scoped>
.how-it-works {
  padding: 72px 24px;
  border-top: 1px solid rgba(255, 255, 255, 0.06);
}

.container {
  max-width: 960px;
  margin: 0 auto;
}

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
  margin: 0 0 48px;
}

.steps {
  display: flex;
  align-items: flex-start;
  gap: 16px;
}

.step {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.step-num {
  width: 32px;
  height: 32px;
  border-radius: 50%;
  background: rgba(0, 229, 255, 0.08);
  border: 1px solid rgba(0, 229, 255, 0.25);
  display: flex;
  align-items: center;
  justify-content: center;
  font-family: var(--vp-font-family-mono);
  font-size: 11px;
  font-weight: 600;
  letter-spacing: 0.05em;
  color: var(--vp-c-brand-1);
  flex-shrink: 0;
}

.step-body h3 {
  font-size: 17px;
  font-weight: 600;
  margin: 0 0 8px;
  color: var(--vp-c-text-1);
}

.step-body p {
  font-size: 14px;
  line-height: 1.65;
  color: var(--vp-c-text-2);
  margin: 0;
}

.step-body code {
  font-family: var(--vp-font-family-mono);
  font-size: 12px;
  background: rgba(255, 255, 255, 0.05);
  padding: 1px 5px;
  border-radius: 2px;
  border: 1px solid rgba(255, 255, 255, 0.08);
  color: #00E5FF;
}

.step-arrow {
  font-size: 20px;
  color: rgba(255, 255, 255, 0.2);
  padding-top: 32px;
  flex-shrink: 0;
}

/* The lit step and the arrow leading into it take the accent. */
.step-num,
.step-arrow {
  transition: color 0.3s ease, background-color 0.3s ease, border-color 0.3s ease, box-shadow 0.3s ease;
}
.step.on .step-num {
  background: rgba(0, 229, 255, 0.18);
  border-color: var(--vp-c-brand-1);
  box-shadow: 0 0 0 4px rgba(0, 229, 255, 0.08);
}
.step-body h3 { transition: color 0.3s ease; }
.step.on h3 { color: var(--vp-c-brand-1); }
.step-arrow.on {
  color: var(--vp-c-brand-1);
  animation: kd-nudge 0.8s ease-in-out;
}
@keyframes kd-nudge {
  50% { translate: 4px 0; }
}
@keyframes kd-nudge-down {
  50% { translate: 0 4px; }
}

@media (max-width: 768px) {
  .steps {
    flex-direction: column;
    gap: 32px;
  }

  .step-arrow.on { animation-name: kd-nudge-down; }

  .step-arrow {
    transform: rotate(90deg);
    padding-top: 0;
    align-self: center;
  }
}
</style>
