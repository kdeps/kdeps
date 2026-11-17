<!--
  Copyright 2026 Kdeps, KvK 94834768
  Licensed under the Apache License, Version 2.0
-->
<template>
  <section class="comparison">
    <div class="container">
      <p class="section-eyebrow">compared to manual code</p>
      <h2 class="section-title">YAML replaces glue code</h2>
      <p class="section-sub">You do not write Python scripts, connection code, or boilerplate.</p>

      <div class="table-wrap">
        <table class="kd-play">
          <thead>
            <tr>
              <th>Traditional approach</th>
              <th>kdeps</th>
            </tr>
          </thead>
          <tbody>
            <tr style="--i: 0">
              <td>Python script + Flask + OpenAI SDK + retry logic</td>
              <td>One <code>workflow.yaml</code> file</td>
            </tr>
            <tr style="--i: 1">
              <td>Dependency code that you write manually</td>
              <td><code>flow.requires:</code> in YAML</td>
            </tr>
            <tr style="--i: 2">
              <td>Error handling that you write manually</td>
              <td><code>onError:</code> block (continue / retry / fail)</td>
            </tr>
            <tr style="--i: 3">
              <td>A Dockerfile and a CI pipeline for deployment</td>
              <td><code>kdeps bundle build --tag</code> then <code>docker push</code></td>
            </tr>
            <tr style="--i: 4">
              <td>Agent chains that you write manually</td>
              <td><code>agent:</code> resource. One agent calls a different agent declaratively</td>
            </tr>
            <tr style="--i: 5">
              <td>A test server that you write</td>
              <td><code>kdeps test</code> runs <code>evals:</code> in the process</td>
            </tr>
            <tr style="--i: 6">
              <td>Handler code that checks the body after the model call</td>
              <td><code>validations.check</code> returns 400 before any model call</td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
  </section>
</template>

<style scoped>
.comparison {
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
  margin: 0 0 24px;
}

.table-wrap {
  overflow-x: auto;
}

table {
  width: 100%;
  border-collapse: collapse;
  border: 1px solid rgba(255, 255, 255, 0.07);
  border-radius: 2px;
  overflow: hidden;
}

th {
  background: rgba(255, 255, 255, 0.05);
  font-weight: 600;
  font-size: 11px;
  letter-spacing: 0.07em;
  text-transform: uppercase;
  color: var(--vp-c-text-2);
  padding: 12px 20px;
  text-align: left;
}

th:last-child {
  color: var(--vp-c-text-1);
  border-top: 2px solid rgba(0, 229, 255, 0.4);
  background: rgba(0, 229, 255, 0.04);
}

td {
  padding: 14px 20px;
  font-size: 14px;
  line-height: 1.55;
  color: var(--vp-c-text-2);
  border-bottom: 1px solid rgba(255, 255, 255, 0.05);
}

td:last-child {
  color: var(--vp-c-text-1);
  font-weight: 500;
}

tr:last-child td {
  border-bottom: none;
}

td code {
  font-family: var(--vp-font-family-mono);
  font-size: 12px;
  background: rgba(255, 255, 255, 0.05);
  padding: 1px 5px;
  border-radius: 2px;
  border: 1px solid rgba(255, 255, 255, 0.08);
  color: #00E5FF;
}

@media (max-width: 768px) {
  .comparison { padding: 48px 16px; }
  td, th { padding: 10px 14px; font-size: 13px; }
}

/* Rows slide in one by one; the manual approach is crossed out, then the
   kdeps answer lights up. */
html.kd-motion table:not(.kd-on) tbody tr { opacity: 0; }
html.kd-motion table.kd-on tbody tr {
  animation: kd-row-in 0.45s cubic-bezier(0.2, 0.8, 0.2, 1) backwards;
  animation-delay: calc(var(--i) * 140ms);
}
html.kd-motion table.kd-on tbody td:first-child {
  text-decoration: line-through;
  text-decoration-color: rgba(255, 107, 129, 0.55);
  animation: kd-cross 0.5s ease backwards;
  animation-delay: calc(var(--i) * 140ms + 450ms);
}
html.kd-motion table.kd-on tbody td:last-child code {
  animation: kd-code-glow 1.2s ease backwards;
  animation-delay: calc(var(--i) * 140ms + 700ms);
}
@keyframes kd-row-in {
  from { opacity: 0; transform: translateX(-12px); }
  to { opacity: 1; transform: none; }
}
@keyframes kd-cross { from { text-decoration-color: transparent; } }
@keyframes kd-code-glow { from { box-shadow: 0 0 0 3px rgba(0, 229, 255, 0.35); } }
</style>
