<!--
  Copyright 2026 Kdeps, KvK 94834768
  Licensed under the Apache License, Version 2.0
-->
<template>
  <section class="comparison">
    <div class="container">
      <p class="section-eyebrow">why kdeps</p>
      <h2 class="section-title">YAML replaces glue code</h2>
      <p class="section-sub">No Python scripts, no wiring, no boilerplate.</p>

      <div class="table-wrap">
        <table>
          <thead>
            <tr>
              <th>Traditional approach</th>
              <th>kdeps</th>
            </tr>
          </thead>
          <tbody>
            <tr>
              <td>Python script + Flask + OpenAI SDK + retry logic</td>
              <td>One <code>workflow.yaml</code> file</td>
            </tr>
            <tr>
              <td>Manual dependency wiring</td>
              <td><code>requires:</code> in YAML</td>
            </tr>
            <tr>
              <td>Write error handling by hand</td>
              <td><code>onError:</code> block (continue / retry / fail)</td>
            </tr>
            <tr>
              <td>Write Dockerfile + CI pipeline for deployment</td>
              <td><code>kdeps bundle build --tag</code> then <code>docker push</code></td>
            </tr>
            <tr>
              <td>Glue code between services</td>
              <td>Resources pass data via <code>output()</code></td>
            </tr>
            <tr>
              <td>Manual polling loop for bots</td>
              <td><code>input.sources: [bot]</code> in workflow config</td>
            </tr>
            <tr>
              <td>Chaining agents by hand</td>
              <td><code>agent:</code> resource — one agent calls another declaratively</td>
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
  margin: 0 0 48px;
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
</style>
