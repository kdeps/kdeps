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

      <div class="diff-block">
        <div class="diff-head">git show HEAD &mdash; resources/llm.yaml</div>
        <pre class="diff"><code><span class="ctx">chat:</span>
<span class="del">-  model: llama3.2:1b</span>
<span class="add">+  model: qwen2.5:7b</span>
<span class="ctx">  prompt: "&lcub;&lcub; get('q') &rcub;&rcub;"</span>
<span class="ctx">validations:</span>
<span class="ctx">  check:</span>
<span class="add">+    - len(get('q')) &lt; 2000        # reject prompts over 2k chars</span></code></pre>
        <p class="diff-caption">One commit, two behavior changes: a bigger model, and the API now 400s on oversized input. No redeploy config, no console toggle - the diff <em>is</em> the change.</p>
      </div>

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
  margin: 0 0 32px;
}

.diff-block {
  border: 1px solid rgba(255, 255, 255, 0.08);
  border-radius: 2px;
  background: rgba(0, 0, 0, 0.25);
  margin: 0 0 40px;
  overflow: hidden;
}

.diff-head {
  font-family: var(--vp-font-family-mono);
  font-size: 11px;
  color: var(--vp-c-text-3);
  padding: 8px 14px;
  border-bottom: 1px solid rgba(255, 255, 255, 0.07);
}

.diff {
  margin: 0;
  padding: 12px 14px;
  overflow-x: auto;
  font-family: var(--vp-font-family-mono);
  font-size: 12.5px;
  line-height: 1.6;
}

.diff .ctx { color: var(--vp-c-text-3); }
.diff .del { color: #ff6b81; display: block; }
.diff .add { color: #00E5FF; display: block; }
.diff .ctx { display: block; }

.diff-caption {
  font-size: 13px;
  color: var(--vp-c-text-2);
  margin: 0;
  padding: 10px 14px 14px;
  border-top: 1px solid rgba(255, 255, 255, 0.07);
}

.diff-caption em {
  color: var(--vp-c-text-1);
  font-style: italic;
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
