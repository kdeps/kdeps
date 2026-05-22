<!--
  Copyright 2026 Kdeps, KvK 94834768
  Licensed under the Apache License, Version 2.0
-->
<template>
  <section class="capabilities">
    <div class="container">
      <h2 class="section-title">Three ways to run</h2>
      <p class="section-sub">Workflows, agents, and agencies — all from the same YAML.</p>

      <div class="cols">
        <div class="col">
          <div class="col-badge workflow">workflow</div>
          <h3>Deterministic pipelines</h3>
          <p>Resources run in DAG order defined by <code>requires:</code>. Every request takes the same path. Predictable, auditable, ships to production.</p>
          <div class="diagram">
            <pre class="d2">
direction: down
A: Request
B: validate
C: llm
D: apiResponse
A -> B -> C -> D
</pre>
          </div>
          <div class="cmd"><span class="prompt">$</span> kdeps run workflow.yaml</div>
          <a href="/modes/workflow-mode" class="learn">Learn more -></a>
        </div>

        <div class="col">
          <div class="col-badge agent">agent</div>
          <h3>Autonomous LLM loop</h3>
          <p>The LLM decides which resources to call and in what order. Every resource auto-registers as a tool. No wiring required.</p>
          <div class="diagram">
            <pre class="d2">
direction: down
A: Prompt
B: LLM
C: Tools
A -> B
B -> C: decides
C -> B: result
B -> D: answer
D: Response
</pre>
          </div>
          <div class="cmd"><span class="prompt">$</span> kdeps serve workflow.yaml</div>
          <a href="/modes/agent-mode" class="learn">Learn more -></a>
        </div>

        <div class="col">
          <div class="col-badge agency">agency</div>
          <h3>Multi-agent orchestration</h3>
          <p>One agent calls another via the <code>agent:</code> resource type. Compose agents like functions — each runs independently, results flow back.</p>
          <div class="diagram">
            <pre class="d2">
direction: down
A: Caller
B: Summariser
C: Translator
D: Response
A -> B
B -> C: agent:
C -> B: result
B -> D
</pre>
          </div>
          <div class="cmd"><span class="prompt">$</span> kdeps run agency.yaml</div>
          <a href="/concepts/agency" class="learn">Learn more -></a>
        </div>
      </div>
    </div>
  </section>
</template>

<style scoped>
.capabilities {
  padding: 72px 24px;
  border-top: 1px solid rgba(255, 255, 255, 0.06);
}

.container {
  max-width: 1024px;
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

.cols {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 20px;
}

.col {
  background: var(--vp-c-bg-soft);
  border: 1px solid rgba(255, 255, 255, 0.07);
  border-radius: 2px;
  padding: 24px;
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.col-badge {
  font-family: var(--vp-font-family-mono);
  font-size: 10px;
  font-weight: 600;
  letter-spacing: 0.1em;
  text-transform: uppercase;
  display: inline-block;
  width: fit-content;
  padding: 2px 8px;
  border-radius: 2px;
  border: 1px solid;
}

.col-badge.workflow { color: #FFD60A; border-color: rgba(255, 214, 10, 0.3); }
.col-badge.agent    { color: #00E5FF; border-color: rgba(0, 229, 255, 0.3); }
.col-badge.agency   { color: #FF2D78; border-color: rgba(255, 45, 120, 0.3); }

.col h3 {
  font-size: 17px;
  font-weight: 600;
  margin: 0;
  color: var(--vp-c-text-1);
}

.col p {
  font-size: 14px;
  line-height: 1.65;
  color: var(--vp-c-text-2);
  margin: 0;
  flex: 1;
}

.col p code {
  font-family: var(--vp-font-family-mono);
  font-size: 12px;
  background: rgba(255, 255, 255, 0.05);
  padding: 1px 5px;
  border-radius: 2px;
  border: 1px solid rgba(255, 255, 255, 0.08);
  color: #00E5FF;
}

.diagram {
  background: rgba(0, 0, 0, 0.25);
  border: 1px solid rgba(255, 255, 255, 0.05);
  border-radius: 2px;
  padding: 12px;
  overflow-x: auto;
}

.d2 {
  font-family: var(--vp-font-family-mono);
  font-size: 10px;
  line-height: 1.5;
  color: rgba(200, 204, 232, 0.5);
  margin: 0;
  white-space: pre;
}

.cmd {
  font-family: var(--vp-font-family-mono);
  font-size: 12px;
  color: rgba(200, 204, 232, 0.55);
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

.learn {
  font-size: 13px;
  font-weight: 500;
  color: var(--vp-c-brand-1);
  text-decoration: none;
  transition: color 0.15s;
  margin-top: auto;
}

.learn:hover { color: #33eaff; }

@media (max-width: 768px) {
  .cols { grid-template-columns: 1fr; }
  .capabilities { padding: 48px 16px; }
}
</style>
