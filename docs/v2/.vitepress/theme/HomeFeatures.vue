<!--
  Copyright 2026 Kdeps, KvK 94834768
  Licensed under the Apache License, Version 2.0
-->
<template>
  <section class="features-custom">
    <div class="container">
      <p class="section-eyebrow">why kdeps</p>
      <h2 class="section-title">Local first, production ready</h2>
      <p class="section-sub">Start on your laptop. Ship to any infra. No rewrites.</p>

      <div class="grid">
        <div class="card">
          <h3>Local AI agent</h3>
          <p>Run <code>kdeps</code> and you are in an AI REPL. Use llamafile or Ollama for a fully offline, private coding agent - no API key, no cloud dependency, no setup.</p>
          <a href="/getting-started/local-agent" class="link">Run locally -></a>
          <a href="/modes/agent-loop-mode" class="link">Agent mode -></a>
        </div>

        <div class="card">
          <h3>Portable workflows</h3>
          <p>Write a <code>workflow.yaml</code> once. Run it locally with <code>kdeps run</code>. Deploy as Docker, Kubernetes, or a binary when you are ready - same file, any target.</p>
          <a href="/modes/workflow-mode" class="link">Workflow mode -></a>
        </div>

        <div class="card">
          <h3>Multi-agent agencies</h3>
          <p>One agent calls another declaratively via the <code>agent:</code> resource type. Each runs independently - results flow back to the caller. Compose agents like functions.</p>
          <a href="/concepts/agency" class="link">Agencies -></a>
        </div>

        <div class="card">
          <h3>Any backend, no lock-in</h3>
          <p>llamafile and Ollama work out of the box - no server install, no API key. Or use OpenAI, Anthropic, Groq, or any OpenAI-compatible endpoint. Apache 2.0 license.</p>
          <a href="/resources/llm-backends" class="link">LLM backends -></a>
        </div>

        <div class="card">
          <h3>Deploy anywhere</h3>
          <p>Export as Docker image, Kubernetes manifests, bootable ISO, or a self-contained single binary. Same workflow, any target. No rewrites.</p>
          <a href="/guides/deployment-guide" class="link">Deployment guide -></a>
        </div>

        <div class="card">
          <h3>Validate and iterate fast</h3>
          <p><code>kdeps validate</code> checks schema, dependencies, and expressions before you ship. <code>kdeps run --dev</code> hot-reloads on every file change.</p>
          <a href="/reference/cli/dev" class="link">CLI reference -></a>
        </div>
      </div>
    </div>
  </section>
</template>

<style scoped>
.features-custom {
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

.grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 16px;
}

.card {
  background: rgba(255, 255, 255, 0.02);
  border: 1px solid rgba(255, 255, 255, 0.07);
  border-radius: 2px;
  padding: 24px;
  display: flex;
  flex-direction: column;
  gap: 10px;
  transition: border-color 0.15s ease, transform 0.15s ease;
}

.card:hover {
  border-color: rgba(0, 229, 255, 0.2);
  transform: translateY(-1px);
}

.card h3 {
  font-size: 17px;
  font-weight: 600;
  color: var(--vp-c-text-1);
  margin: 0;
}

.card p {
  font-size: 14px;
  line-height: 1.65;
  color: var(--vp-c-text-2);
  margin: 0;
  flex: 1;
}

.card p code {
  font-family: var(--vp-font-family-mono);
  font-size: 12px;
  background: rgba(255, 255, 255, 0.05);
  padding: 1px 5px;
  border-radius: 2px;
  border: 1px solid rgba(255, 255, 255, 0.08);
  color: #00E5FF;
}

.link {
  font-size: 13px;
  font-weight: 500;
  color: rgba(0, 229, 255, 0.7);
  text-decoration: none;
  transition: color 0.15s;
}

.link:hover { color: #33eaff; }

@media (max-width: 768px) {
  .features-custom { padding: 48px 16px; }
  .grid { grid-template-columns: 1fr; }
}
</style>
