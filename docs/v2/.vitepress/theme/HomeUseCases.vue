<!--
  Copyright 2026 Kdeps, KvK 94834768
  Licensed under the Apache License, Version 2.0
-->
<template>
  <section class="use-cases">
    <div class="container">
      <h2 class="section-title">What you can build</h2>
      <p class="section-sub">Real patterns from the examples directory. Every one is a working workflow.</p>

      <div class="cards">
        <a href="/examples/showcase" class="card">
          <h3>REST API</h3>
          <p>POST a JSON body, run a DAG pipeline, get structured JSON back. The default pattern for workflow mode.</p>
          <span class="mode-tag">workflow mode</span>
        </a>

        <a href="/examples/telegram-bot/" class="card">
          <h3>Telegram Bot</h3>
          <p>Poll for messages, run a multi-step pipeline, reply with LLM responses. Two resources: llm and reply.</p>
          <span class="mode-tag">workflow mode</span>
        </a>

        <a href="/examples/stateless-bot/" class="card">
          <h3>Stateless Bot</h3>
          <p>Read from stdin, call an LLM, write to stdout. One-shot. Perfect for cron jobs and CI pipelines.</p>
          <span class="mode-tag">workflow mode</span>
        </a>

        <a href="/resources/embedding" class="card">
          <h3>RAG Search</h3>
          <p>Index documents locally, search with keywords, feed results into an LLM prompt. Fully on-prem.</p>
          <span class="mode-tag">workflow mode</span>
        </a>

        <a href="/concepts/agency" class="card">
          <h3>Agency Pipeline</h3>
          <p>Orchestrate multiple agents. One summarises, another translates — each an independent workflow.</p>
          <span class="mode-tag agency-tag">multi-agent</span>
        </a>
      </div>
    </div>
  </section>
</template>

<style scoped>
.use-cases {
  padding: 72px 24px 96px;
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

.cards {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(240px, 1fr));
  gap: 16px;
}

.card {
  background: var(--vp-c-bg-soft);
  border: 1px solid rgba(255, 255, 255, 0.07);
  border-radius: 2px;
  padding: 20px;
  display: flex;
  flex-direction: column;
  gap: 10px;
  text-decoration: none;
  transition: transform 0.15s ease, border-color 0.15s ease;
}

.card:hover {
  transform: translateY(-2px);
  border-color: rgba(0, 229, 255, 0.3);
}

.card h3 {
  font-size: 16px;
  font-weight: 600;
  color: var(--vp-c-text-1);
  margin: 0;
}

.card p {
  font-size: 13px;
  line-height: 1.6;
  color: var(--vp-c-text-2);
  margin: 0;
  flex: 1;
}

.mode-tag {
  font-family: var(--vp-font-family-mono);
  font-size: 10px;
  font-weight: 600;
  letter-spacing: 0.08em;
  text-transform: uppercase;
  color: var(--vp-c-text-3);
  align-self: flex-start;
  border: 1px solid rgba(255, 255, 255, 0.08);
  border-radius: 2px;
  padding: 2px 8px;
}

.agency-tag {
  color: #FF2D78;
  border-color: rgba(255, 45, 120, 0.3);
}

@media (max-width: 768px) {
  .use-cases { padding: 48px 16px 72px; }
  .cards { grid-template-columns: 1fr; }
}
</style>
