<!--
  Copyright 2026 Kdeps, KvK 94834768
  Licensed under the Apache License, Version 2.0
-->
<template>
  <section class="registry">
    <div class="container">
      <p class="section-eyebrow">registry</p>
      <h2 class="section-title">Components, workflows, and agencies</h2>
      <p class="section-sub">Install pre-built packages from the registry. Publish your own.</p>

      <div class="cols">
        <div class="col">
          <div class="col-badge">components</div>
          <h3>Components</h3>
          <p>Reusable capability extensions. Install with one command, invoke with <code>component:</code> and typed inputs.</p>
          <div class="packages">
            <span class="pkg">scraper</span>
            <span class="pkg">search</span>
            <span class="pkg">browser</span>
            <span class="pkg">embedding</span>
            <span class="pkg">tts</span>
            <span class="pkg">email</span>
          </div>
          <div class="cmd"><span class="prompt">$</span> kdeps registry install scraper</div>
        </div>

        <div class="col">
          <div class="col-badge">workflows</div>
          <h3>Workflows</h3>
          <p>Complete DAG pipelines packaged as <code>.kdeps</code> archives. Drop them into an agency or run them standalone.</p>
          <div class="packages">
            <span class="pkg">summarizer</span>
            <span class="pkg">classifier</span>
            <span class="pkg">translator</span>
            <span class="pkg">invoice-extractor</span>
          </div>
          <div class="cmd"><span class="prompt">$</span> kdeps run summarizer-1.0.0.kdeps</div>
        </div>

        <div class="col">
          <div class="col-badge">agencies</div>
          <h3>Agencies</h3>
          <p>Multi-agent orchestration bundles packaged as <code>.kagency</code> archives. One entry point, many agents.</p>
          <div class="packages">
            <span class="pkg">cv-matcher</span>
            <span class="pkg">research-pipeline</span>
            <span class="pkg">content-agency</span>
          </div>
          <div class="cmd"><span class="prompt">$</span> kdeps run cv-matcher-1.0.0.kagency</div>
        </div>
      </div>

      <div class="cta-row">
        <a href="https://kdeps.io" class="cta">Browse registry -></a>
        <a href="/reference/cli/registry" class="cta">Publish a package -></a>
      </div>
    </div>
  </section>
</template>

<style scoped>
.registry {
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

.cols {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 16px;
  margin-bottom: 32px;
}

.col {
  background: rgba(255, 255, 255, 0.02);
  border: 1px solid rgba(255, 255, 255, 0.07);
  border-radius: 2px;
  padding: 24px;
  display: flex;
  flex-direction: column;
  gap: 12px;
  transition: border-color 0.15s ease, transform 0.15s ease;
}

.col:hover {
  border-color: rgba(0, 229, 255, 0.2);
  transform: translateY(-1px);
}

.col-badge {
  font-family: var(--vp-font-family-mono);
  font-size: 10px;
  font-weight: 600;
  letter-spacing: 0.1em;
  text-transform: uppercase;
  color: var(--vp-c-brand-1);
}

.col h3 {
  font-size: 17px;
  font-weight: 600;
  color: var(--vp-c-text-1);
  margin: 0;
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

.packages {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.pkg {
  font-family: var(--vp-font-family-mono);
  font-size: 11px;
  color: rgba(200, 204, 232, 0.5);
  background: rgba(255, 255, 255, 0.04);
  border: 1px solid rgba(255, 255, 255, 0.06);
  border-radius: 2px;
  padding: 2px 8px;
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

.cta-row {
  display: flex;
  justify-content: center;
  gap: 24px;
}

.cta {
  font-size: 14px;
  font-weight: 500;
  color: var(--vp-c-brand-1);
  text-decoration: none;
  transition: color 0.15s;
}

.cta:hover { color: #33eaff; }

@media (max-width: 768px) {
  .registry { padding: 48px 16px; }
  .cols { grid-template-columns: 1fr; }
  .cta-row { flex-direction: column; align-items: center; gap: 12px; }
}
</style>
