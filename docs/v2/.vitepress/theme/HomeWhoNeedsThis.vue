<!--
  Copyright 2026 Kdeps, KvK 94834768
  Licensed under the Apache License, Version 2.0
-->
<template>
  <section class="who-needs">
    <div class="container">
      <p class="section-eyebrow">who it's for</p>
      <h2 class="section-title">When you cannot send data to an external AI service</h2>
      <p class="section-sub">kdeps packages the model with the workflow. As a result, all of it runs in your boundary.</p>

      <div class="kd-play cards">
        <a href="/deploy/kubernetes" class="kd-stagger card" style="--i: 0">
          <h3>Private enterprise AI</h3>
          <p>Deploy an internal RAG or document agent fully in the VPC of your customer. No prompts, documents, or embeddings go out of their infrastructure.</p>
          <span class="tag">on-prem &middot; regulated</span>
        </a>

        <a href="/deploy/binaries" class="kd-stagger card" style="--i: 1">
          <h3>Edge &amp; air-gapped AI</h3>
          <p>Ship the model and the workflow as a bootable ISO or one binary. Write the ISO to a USB stick. It runs on a disconnected device, with no network at inference time.</p>
          <span class="tag">edge &middot; offline</span>
        </a>

        <a href="/workflow/quickstart" class="kd-stagger card" style="--i: 2">
          <h3>AI APIs without glue code</h3>
          <p>Change a YAML workflow into a production HTTP API and container. You do not need FastAPI, LangChain code, or a Dockerfile that you write manually. You declare the validation, the order, and the response shape.</p>
          <span class="tag">workflow</span>
        </a>
      </div>

      <p class="build-label">Working examples. Each example is a project that you can run:</p>
      <div class="kd-play chips">
        <a href="/examples/file-processor" class="kd-stagger chip" style="--i: 0">Document summarizer</a>
        <a href="/examples/rag-search" class="kd-stagger chip" style="--i: 1">RAG over your docs</a>
        <a href="/examples/function-calling" class="kd-stagger chip" style="--i: 2">Function calling</a>
        <a href="/examples/sql-api" class="kd-stagger chip" style="--i: 3">SQL-backed API</a>
        <a href="/examples/telegram-bot/" class="kd-stagger chip" style="--i: 4">Telegram bot</a>
        <a href="/examples/agency" class="kd-stagger chip" style="--i: 5">Multi-agent agency</a>
        <a href="/examples/" class="kd-stagger chip chip-more" style="--i: 6">All examples</a>
      </div>
    </div>
  </section>
</template>

<style scoped>
.who-needs {
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
  grid-template-columns: repeat(auto-fill, minmax(260px, 1fr));
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
  border-color: rgba(0, 229, 255, 0.25);
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

.tag {
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

.build-label {
  font-size: 13px;
  color: var(--vp-c-text-3);
  margin: 40px 0 12px;
}

.chips {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.chip {
  font-size: 13px;
  color: var(--vp-c-text-2);
  text-decoration: none;
  border: 1px solid rgba(255, 255, 255, 0.08);
  border-radius: 2px;
  padding: 5px 10px;
  transition: border-color 0.15s ease, color 0.15s ease;
}

.chip:hover {
  border-color: rgba(0, 229, 255, 0.3);
  color: var(--vp-c-text-1);
}

.chip-more {
  color: var(--vp-c-brand-1);
}

@media (max-width: 768px) {
  .who-needs { padding: 48px 16px; }
  .cards { grid-template-columns: 1fr; }
}

/* Each card's tag pops in just after its card lands. */
html.kd-motion .kd-play:not(.kd-on) .tag { opacity: 0; }
html.kd-motion .kd-play.kd-on .tag {
  animation: kd-tag-pop 0.4s cubic-bezier(0.2, 1.5, 0.4, 1) backwards;
  animation-delay: calc(var(--i) * 90ms + 350ms);
}
@keyframes kd-tag-pop { from { opacity: 0; transform: scale(0.7); } }
</style>
