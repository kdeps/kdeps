<!--
  Copyright 2026 Kdeps, KvK 94834768
  Licensed under the Apache License, Version 2.0
-->
<template>
  <section class="features-custom">
    <div class="container">
      <p class="section-eyebrow">why kdeps</p>
      <h2 class="section-title">YAML replaces glue code</h2>
      <p class="section-sub">No Python scripts, no wiring, no boilerplate.</p>

      <div class="grid">
        <div class="card">
          <h3>Two modes, one file</h3>
          <p>Workflow mode runs deterministic DAG pipelines via <code>kdeps run</code>. Agent mode starts an autonomous LLM loop via <code>kdeps serve</code>. Same <code>workflow.yaml</code> works in both.</p>
          <a href="/modes/workflow-mode" class="link">Workflow mode -></a>
          <a href="/modes/agent-loop-mode" class="link">Agent mode -></a>
        </div>

        <div class="card">
          <h3>Multi-agent agencies</h3>
          <p>One agent calls another declaratively via the <code>agent:</code> resource type. Each runs independently — results flow back to the caller. Compose agents like functions.</p>
          <a href="/concepts/agency" class="link">Agencies -></a>
        </div>

        <div class="card">
          <h3>Any backend, no lock-in</h3>
          <p>Ollama, OpenAI, Anthropic, Groq, or any OpenAI-compatible endpoint. Switch backends in <code>~/.kdeps/config.yaml</code> without touching workflow files. Apache 2.0 license.</p>
          <a href="/resources/llm-backends" class="link">LLM backends -></a>
        </div>

        <div class="card">
          <h3>Component registry</h3>
          <p>Install pre-built scraper, search, browser, and embedding components from the registry. Compose them into your workflow with <code>component:</code> and one line.</p>
          <a href="https://kdeps.io" class="link">Browse registry -></a>
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
