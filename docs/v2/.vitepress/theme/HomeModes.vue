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
          <pre class="mode-diagram">POST /summarize
      |
      v
 +---------+
 |  fetch  |  httpClient
 +---------+
      |
      v
 +-----------+
 | summarize |  chat
 +-----------+
      |
      v
  apiResponse</pre>
          <div class="mode-cmd"><span class="prompt">$</span> kdeps run workflow.yaml</div>
        </div>

        <!-- Agent mode -->
        <div class="mode-card">
          <div class="mode-header">
            <span class="mode-tag">agent</span>
            <h3>Autonomous LLM loop</h3>
          </div>
          <p class="mode-desc">The LLM decides which resources to call and in what order. Every resource auto-registers as a tool. Multi-step reasoning, no wiring required.</p>
          <pre class="mode-diagram">stdin prompt
      |
      v
 +---------+
 |   LLM   |  plans steps
 +---------+
   |  |  |
   v  v  v
 http sql python  <-- tools
      |
      v
 +---------+
 |   LLM   |  synthesizes
 +---------+
      |
      v
   response</pre>
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

.mode-diagram {
  font-family: var(--vp-font-family-mono);
  font-size: 11px;
  line-height: 1.5;
  color: rgba(200, 204, 232, 0.5);
  background: rgba(0, 0, 0, 0.3);
  border: 1px solid rgba(255, 255, 255, 0.05);
  border-radius: 2px;
  padding: 12px 14px;
  margin: 0;
  white-space: pre;
  overflow-x: auto;
  flex: 1;
}

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
}
</style>
