<!--
  Copyright 2026 Kdeps, KvK 94834768
  Licensed under the Apache License, Version 2.0
-->
<template>
  <section class="use-cases">
    <div class="container">
      <p class="section-eyebrow">examples</p>
      <h2 class="section-title">What you can build</h2>
      <p class="section-sub">Every card links to a working project in the examples directory.</p>

      <div class="cards">
        <a href="/examples/file-processor" class="card">
          <h3>Document summarizer</h3>
          <p>POST a file, get back a structured JSON summary - title, key points, entities. Runs on a local model.</p>
          <span class="mode-tag">workflow mode</span>
        </a>

        <a href="/examples/rag-search" class="card">
          <h3>RAG over your own docs</h3>
          <p>Index PDFs and text locally, retrieve the relevant chunks, and answer with an LLM. Nothing leaves the box.</p>
          <span class="mode-tag">workflow mode</span>
        </a>

        <a href="/examples/web-scraper" class="card">
          <h3>Web scraper API</h3>
          <p>Fetch a page, extract the fields you name, return clean JSON. Swap in the browser resource for JS-heavy sites.</p>
          <span class="mode-tag">workflow mode</span>
        </a>

        <a href="/examples/sql-api" class="card">
          <h3>SQL-backed API</h3>
          <p>Turn a plain-language question into a query against your database and answer it. Parameterized, not string-built.</p>
          <span class="mode-tag">workflow mode</span>
        </a>

        <a href="/examples/telephony-bot" class="card">
          <h3>Phone assistant (IVR)</h3>
          <p>Answer a call, transcribe the caller, run a pipeline, speak the reply - say, ask, and menu steps in YAML.</p>
          <span class="mode-tag">workflow mode</span>
        </a>

        <a href="/examples/agency" class="card">
          <h3>Multi-agent agency</h3>
          <p>One agent researches, another writes - each its own workflow.yaml, composed like function calls.</p>
          <span class="mode-tag agency-tag">multi-agent</span>
        </a>
      </div>
    </div>
  </section>
</template>

<style scoped>
.use-cases {
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

.card:has(.mode-tag:not(.agency-tag)):hover {
  transform: translateY(-2px);
  border-color: rgba(255, 214, 10, 0.25);
}

.card:has(.agency-tag):hover {
  transform: translateY(-2px);
  border-color: rgba(255, 45, 120, 0.25);
}

.card h3 {
  font-size: 16px;
  font-weight: 600;
  color: var(--vp-c-text-1);
  margin: 0;
}

.card p {
  font-size: 14px;
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
  .use-cases { padding: 48px 16px; }
  .cards { grid-template-columns: 1fr; }
}
</style>
