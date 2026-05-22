<!--
  Copyright 2026 Kdeps, KvK 94834768
  Licensed under the Apache License, Version 2.0
-->
<template>
  <section class="how-it-works">
    <div class="container">
      <h2 class="section-title">How it works</h2>
      <p class="section-sub">Three steps from idea to running AI API.</p>

      <div class="steps">
        <div class="step">
          <div class="step-num">01</div>
          <div class="step-body">
            <h3>Write YAML</h3>
            <p>Declare resources -- chat, HTTP, Python, SQL, shell. Wire them with <code>requires:</code>. No glue code.</p>
          </div>
        </div>

        <div class="step-arrow">→</div>

        <div class="step">
          <div class="step-num">02</div>
          <div class="step-body">
            <h3>Run kdeps</h3>
            <p><code>kdeps run workflow.yaml</code> starts the API server. <code>kdeps serve</code> starts the autonomous agent loop.</p>
          </div>
        </div>

        <div class="step-arrow">→</div>

        <div class="step">
          <div class="step-num">03</div>
          <div class="step-body">
            <h3>Call the API</h3>
            <p>POST to your route, get back structured JSON. Export as Docker, Kubernetes, ISO, or a single binary.</p>
          </div>
        </div>
      </div>
    </div>
  </section>
</template>

<style scoped>
.how-it-works {
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

.steps {
  display: flex;
  align-items: flex-start;
  gap: 16px;
}

.step {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.step-num {
  font-family: var(--vp-font-family-mono);
  font-size: 11px;
  font-weight: 600;
  letter-spacing: 0.1em;
  color: var(--vp-c-brand-1);
}

.step-body h3 {
  font-size: 17px;
  font-weight: 600;
  margin: 0 0 8px;
  color: var(--vp-c-text-1);
}

.step-body p {
  font-size: 14px;
  line-height: 1.65;
  color: var(--vp-c-text-2);
  margin: 0;
}

.step-body code {
  font-family: var(--vp-font-family-mono);
  font-size: 12px;
  background: rgba(255, 255, 255, 0.05);
  padding: 1px 5px;
  border-radius: 2px;
  border: 1px solid rgba(255, 255, 255, 0.08);
  color: #00E5FF;
}

.step-arrow {
  font-size: 20px;
  color: rgba(255, 255, 255, 0.2);
  padding-top: 32px;
  flex-shrink: 0;
}

@media (max-width: 768px) {
  .steps {
    flex-direction: column;
    gap: 32px;
  }

  .step-arrow {
    transform: rotate(90deg);
    padding-top: 0;
    align-self: center;
  }
}
</style>
