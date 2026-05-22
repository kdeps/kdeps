<!--
  Copyright 2026 Kdeps, KvK 94834768
  Licensed under the Apache License, Version 2.0
-->
<template>
  <section class="modes-section">
    <div class="container">
      <h2 class="section-title">Two modes</h2>
      <p class="section-sub">Pick the one that fits the task. Mix them in an agency.</p>

      <div class="modes">
        <!-- Workflow mode -->
        <div class="mode-card">
          <div class="mode-header">
            <span class="mode-tag">workflow</span>
            <h3>Deterministic pipelines</h3>
          </div>
          <p class="mode-desc">Resources run in DAG order defined by <code>requires:</code>. Every request takes the same path. Predictable, testable, auditable.</p>

          <div class="flow">
            <div class="flow-node entry">POST /summarize</div>
            <div class="flow-arrow">↓</div>
            <div class="flow-node">
              <span class="node-name">fetch</span>
              <span class="node-type">httpClient</span>
            </div>
            <div class="flow-arrow">↓</div>
            <div class="flow-node">
              <span class="node-name">summarize</span>
              <span class="node-type">chat</span>
            </div>
            <div class="flow-arrow">↓</div>
            <div class="flow-node exit">apiResponse</div>
          </div>

          <div class="mode-cmd"><span class="prompt">$</span> kdeps run workflow.yaml</div>
        </div>

        <!-- Agent mode -->
        <div class="mode-card">
          <div class="mode-header">
            <span class="mode-tag">agent</span>
            <h3>Autonomous LLM loop</h3>
          </div>
          <p class="mode-desc">The LLM decides which resources to call and in what order. Every resource auto-registers as a tool. Multi-step reasoning, no wiring required.</p>

          <div class="flow">
            <div class="flow-node entry">stdin prompt</div>
            <div class="flow-arrow">↓</div>
            <div class="flow-node llm">
              <span class="node-name">LLM</span>
              <span class="node-type">plans steps</span>
            </div>
            <div class="flow-arrow">↓</div>
            <div class="flow-tools">
              <div class="tool-node">http</div>
              <div class="tool-node">sql</div>
              <div class="tool-node">python</div>
            </div>
            <div class="flow-arrow">↓</div>
            <div class="flow-node llm">
              <span class="node-name">LLM</span>
              <span class="node-type">synthesizes</span>
            </div>
            <div class="flow-arrow">↓</div>
            <div class="flow-node exit">response</div>
          </div>

          <div class="mode-cmd"><span class="prompt">$</span> kdeps serve workflow.yaml</div>
        </div>
      </div>
    </div>
  </section>
</template>

<style scoped>
.modes-section {
  padding: 72px 24px;
  border-top: 1px solid rgba(255, 255, 255, 0.06);
}

.container {
  max-width: 960px;
  margin: 0 auto;
}

.section-title {
  font-size: 28px;
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

.modes {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 24px;
}

.mode-card {
  background: var(--vp-c-bg-soft);
  border: 1px solid rgba(255, 255, 255, 0.07);
  border-top: 2px solid var(--vp-c-brand-1);
  border-radius: 2px;
  padding: 24px;
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.mode-header {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.mode-tag {
  font-family: var(--vp-font-family-mono);
  font-size: 10px;
  font-weight: 600;
  letter-spacing: 0.1em;
  text-transform: uppercase;
  color: var(--vp-c-brand-1);
}

.mode-header h3 {
  font-size: 18px;
  font-weight: 600;
  margin: 0;
  color: var(--vp-c-text-1);
}

.mode-desc {
  font-size: 14px;
  line-height: 1.65;
  color: var(--vp-c-text-2);
  margin: 0;
}

.mode-desc code {
  font-family: var(--vp-font-family-mono);
  font-size: 12px;
  background: rgba(255, 255, 255, 0.05);
  padding: 1px 5px;
  border-radius: 2px;
  border: 1px solid rgba(255, 255, 255, 0.08);
  color: #00E5FF;
}

/* --- flowchart --- */
.flow {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 0;
  background: rgba(0, 0, 0, 0.25);
  border: 1px solid rgba(255, 255, 255, 0.05);
  border-radius: 2px;
  padding: 16px 12px;
  flex: 1;
}

.flow-node {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  width: 100%;
  max-width: 200px;
  background: #141414;
  border: 1px solid #2a2a2a;
  border-radius: 2px;
  padding: 6px 12px;
  font-family: var(--vp-font-family-mono);
  font-size: 11px;
  color: rgba(200, 204, 232, 0.75);
  text-align: center;
}

.flow-node.entry,
.flow-node.exit {
  border-color: var(--vp-c-brand-1);
  color: var(--vp-c-brand-1);
}

.flow-node.llm {
  border-color: rgba(0, 229, 255, 0.35);
}

.node-name {
  color: rgba(200, 204, 232, 0.9);
  font-weight: 500;
}

.node-type {
  color: rgba(200, 204, 232, 0.35);
  font-size: 10px;
}

.flow-arrow {
  font-size: 14px;
  color: var(--vp-c-brand-1);
  line-height: 1;
  padding: 2px 0;
  user-select: none;
}

.flow-tools {
  display: flex;
  gap: 6px;
  justify-content: center;
  width: 100%;
}

.tool-node {
  background: #141414;
  border: 1px solid #2a2a2a;
  border-radius: 2px;
  padding: 4px 10px;
  font-family: var(--vp-font-family-mono);
  font-size: 10px;
  color: rgba(200, 204, 232, 0.55);
}

/* --- terminal command --- */
.mode-cmd {
  font-family: var(--vp-font-family-mono);
  font-size: 12px;
  color: rgba(200, 204, 232, 0.6);
  background: rgba(0, 0, 0, 0.2);
  border: 1px solid rgba(255, 255, 255, 0.05);
  border-radius: 2px;
  padding: 8px 12px;
}

.prompt {
  color: var(--vp-c-brand-1);
  margin-right: 8px;
  user-select: none;
}

@media (max-width: 768px) {
  .modes {
    grid-template-columns: 1fr;
  }
  .modes-section {
    padding: 48px 16px;
  }
}
</style>
