<!--
  Copyright 2026 Kdeps, KvK 94834768
  Licensed under the Apache License, Version 2.0
-->
<template>
  <section class="git-native">
    <div class="container">
      <p class="section-eyebrow">git-native</p>
      <h2 class="section-title">The YAML is the behavior spec</h2>
      <p class="section-sub">The YAML in your repository defines what the appliance does. A commit changes the behavior, and git history is its changelog.</p>

      <div class="diff-block kd-play">
        <div class="diff-head">git show HEAD - resources/llm.yaml</div>
        <pre class="diff"><code><span class="ctx" style="--i: 0">validations:</span>
<span class="ctx" style="--i: 1">  check:</span>
<span class="ctx" style="--i: 2">    - get('q') != ''</span>
<span class="add" style="--i: 3">+    - len(get('q')) &lt; 2000      # reject prompts over 2000 characters</span>
<span class="ctx" style="--i: 4">chat:</span>
<span class="ctx" style="--i: 5">  model:</span>
<span class="del" style="--i: 6">-    name: llama3.2:1b</span>
<span class="add" style="--i: 7">+    name: qwen2.5:7b</span>
<span class="ctx" style="--i: 8">  context:</span>
<span class="ctx" style="--i: 9">    prompt: "&lcub;&lcub; get('q') &rcub;&rcub;"</span></code></pre>
        <p class="diff-caption">One commit, two behavior changes. The diff <em>is</em> the change.</p>
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
.diff .del { color: #ff6b81; }
.diff .add { color: #00E5FF; }

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


@media (max-width: 768px) {
  .git-native { padding: 48px 16px; }
}

/* The diff plays like git show: lines print in order, the removed line is
   struck through, and added lines flash before settling. */
.diff .add { background: rgba(0, 229, 255, 0.08); }
.diff .del { text-decoration: line-through; text-decoration-color: rgba(255, 107, 129, 0.6); }
html.kd-motion .diff-block:not(.kd-on) .diff span,
html.kd-motion .diff-block:not(.kd-on) .diff-caption { opacity: 0; }
html.kd-motion .diff-block.kd-on .diff span {
  animation: kd-diff-in 0.3s ease backwards;
  animation-delay: calc(var(--i) * 110ms);
}
html.kd-motion .diff-block.kd-on .diff .add {
  animation: kd-diff-in 0.3s ease backwards, kd-add-flash 1.4s ease backwards;
  animation-delay: calc(var(--i) * 110ms), calc(var(--i) * 110ms + 250ms);
}
html.kd-motion .diff-block.kd-on .diff .del {
  animation: kd-diff-in 0.3s ease backwards, kd-strike 0.5s ease backwards;
  animation-delay: calc(var(--i) * 110ms), calc(var(--i) * 110ms + 300ms);
}
html.kd-motion .diff-block.kd-on .diff-caption { animation: kd-diff-in 0.6s ease 1.3s backwards; }
@keyframes kd-diff-in { from { opacity: 0; } }
@keyframes kd-add-flash { from { background: rgba(0, 229, 255, 0.35); } }
@keyframes kd-strike { from { text-decoration-color: transparent; } }
</style>
