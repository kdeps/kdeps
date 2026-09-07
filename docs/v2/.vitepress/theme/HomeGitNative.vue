<!--
  Copyright 2026 Kdeps, KvK 94834768
  Licensed under the Apache License, Version 2.0
-->
<template>
  <section class="git-native">
    <div class="container">
      <p class="section-eyebrow">git-native</p>
      <h2 class="section-title">The YAML is the behavior spec</h2>
      <p class="section-sub">What the appliance does - model, validation, step order, response shape - is defined entirely by the YAML in your repo. Change it, commit, and it behaves differently; your git history is the changelog of the agent's behavior. Only the model binding lives outside the repo, so the same commit runs local on a laptop and cloud in production.</p>

      <div class="cards">
        <div class="card">
          <h3>Reviewable</h3>
          <p>A change to an agent is a diff: a new resource, a tightened validation, a reshaped response. Review it in a pull request like any other code.</p>
        </div>
        <div class="card">
          <h3>Versioned</h3>
          <p>A git tag is a release. <code>kdeps validate</code> and <code>kdeps bundle build</code> run against any checkout or tag; the built appliance pins the runtime and the model too.</p>
        </div>
        <div class="card">
          <h3>Reproducible</h3>
          <p>Point CI at a tag: <code>kdeps validate</code>, <code>kdeps bundle build</code>, push the image. The same commit produces the same appliance every time.</p>
        </div>
        <div class="card">
          <h3>Shareable, optionally</h3>
          <p>The <a href="/registry/">registry</a> is a convenience for installing and publishing agents by name - <code>kdeps registry install owner/repo</code> pulls from a GitHub repo. You never need it to build or deploy your own.</p>
        </div>
      </div>
    </div>
  </section>
</template>

<style scoped>
.git-native {
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
  grid-template-columns: repeat(auto-fill, minmax(210px, 1fr));
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
}

.card code {
  font-family: var(--vp-font-family-mono);
  font-size: 12px;
  background: rgba(255, 255, 255, 0.05);
  padding: 1px 5px;
  border-radius: 2px;
  border: 1px solid rgba(255, 255, 255, 0.08);
  color: #00E5FF;
  word-break: break-word;
}

@media (max-width: 768px) {
  .git-native { padding: 48px 16px; }
  .cards { grid-template-columns: 1fr; }
}
</style>
