<!--
  Copyright 2026 Kdeps, KvK 94834768
  Licensed under the Apache License, Version 2.0
-->
<template>
  <section class="backends">
    <div class="container">
      <p class="section-eyebrow">platform agnostic</p>
      <h2 class="section-title">Works with any LLM backend</h2>
      <p class="section-sub">Switch backends in config. No code changes, no lock-in.</p>

      <div class="rows">
        <div class="row" data-row="llm">
          <span class="row-label">LLM Backends</span>
          <div class="tiles">
            <div class="tile" v-for="item in llmBackends" :key="item.name">
              <span class="tile-name">{{ item.name }}</span>
              <span class="tile-sub llm-sub">{{ item.sub }}</span>
            </div>
          </div>
        </div>

        <div class="row" data-row="deploy">
          <span class="row-label">Deploy Targets</span>
          <div class="tiles">
            <div class="tile" v-for="item in deployTargets" :key="item.name">
              <span class="tile-name">{{ item.name }}</span>
              <span class="tile-sub deploy-sub">{{ item.sub }}</span>
            </div>
          </div>
        </div>
      </div>
    </div>
  </section>
</template>

<script setup>
const llmBackends = [
  { name: 'Ollama', sub: 'local' },
  { name: 'OpenAI', sub: 'cloud' },
  { name: 'Anthropic', sub: 'cloud' },
  { name: 'Groq', sub: 'cloud' },
  { name: 'LM Studio', sub: 'local' },
]

const deployTargets = [
  { name: 'Docker', sub: 'docker image' },
  { name: 'Kubernetes', sub: 'helm chart' },
  { name: 'Binary', sub: 'single file' },
  { name: 'ISO', sub: 'bootable image' },
  { name: 'WebServer', sub: 'http server' },
]
</script>

<style scoped>
.backends {
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

.rows {
  display: flex;
  flex-direction: column;
  gap: 32px;
}

.row {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.row-label {
  font-family: var(--vp-font-family-mono);
  font-size: 10px;
  font-weight: 600;
  letter-spacing: 0.1em;
  text-transform: uppercase;
  color: var(--vp-c-text-3);
}

.tiles {
  display: grid;
  grid-template-columns: repeat(5, 1fr);
  gap: 12px;
}

.tile {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 6px;
  padding: 20px 16px;
  background: rgba(255, 255, 255, 0.03);
  border: 1px solid rgba(255, 255, 255, 0.07);
  border-radius: 2px;
  transition: border-color 0.15s ease, transform 0.15s ease;
  cursor: default;
}

[data-row="llm"] .tile:hover {
  border-color: rgba(0, 229, 255, 0.25);
  transform: translateY(-2px);
}

[data-row="deploy"] .tile:hover {
  border-color: rgba(255, 214, 10, 0.25);
  transform: translateY(-2px);
}

.tile-name {
  font-size: 14px;
  font-weight: 600;
  color: var(--vp-c-text-1);
}

.tile-sub {
  font-size: 11px;
  font-family: var(--vp-font-family-mono);
}

.llm-sub {
  color: rgba(0, 229, 255, 0.5);
}

.deploy-sub {
  color: rgba(255, 214, 10, 0.5);
}

@media (max-width: 768px) {
  .backends { padding: 48px 16px; }
  .tiles { grid-template-columns: repeat(3, 1fr); }
}

@media (max-width: 480px) {
  .tiles { grid-template-columns: repeat(2, 1fr); }
}
</style>
