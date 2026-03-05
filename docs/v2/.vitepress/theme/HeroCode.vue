<!--
  Copyright 2026 Kdeps, KvK 94834768
  Licensed under the Apache License, Version 2.0
-->
<template>
  <div class="hero-code-window">
    <!-- Title bar -->
    <div class="code-titlebar">
      <div class="code-dots">
        <span class="r"></span>
        <span class="y"></span>
        <span class="g"></span>
      </div>
      <span class="code-filename">workflow.yaml</span>
    </div>

    <!-- Code body -->
    <pre class="code-body" v-pre><code><span class="key">run</span><span class="op">:</span>
  <span class="key">chat</span><span class="op">:</span>
    <span class="key">model</span><span class="op">:</span> <span class="str">llama3.2:1b</span>
    <span class="key">prompt</span><span class="op">:</span> <span class="str">"Summarize: {{ get('q') }}"</span>

  <span class="key">apiResponse</span><span class="op">:</span>
    <span class="key">success</span><span class="op">:</span> <span class="key">true</span>
    <span class="key">response</span><span class="op">:</span>
      <span class="key">data</span><span class="op">:</span> <span class="str">"{{ get('chat') }}"</span>

<span class="cm"># No glue code. No legacy code.</span></code></pre>

    <!-- Status bar -->
    <div class="code-statusbar">
      <span class="status-dot"></span>
      <span>running · port 16395</span>
      <span>startup: 0.1s</span>
    </div>
  </div>
</template>

<style scoped>
.hero-code-window {
  font-family: var(--vp-font-family-mono);
  font-size: 13px;
  line-height: 1.6;
  background: #070707;
  border: 1px solid rgba(255, 255, 255, 0.08);
  border-left: 2px solid var(--vp-c-brand-1);
  border-radius: 2px;
  overflow: hidden;
  width: 100%;
  max-width: 480px;
  box-shadow: var(--vp-shadow-3);
}

/* Title bar */
.code-titlebar {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 14px;
  background: rgba(0, 229, 255, 0.04);
  border-bottom: 1px solid rgba(255, 255, 255, 0.06);
}

.code-dots {
  display: flex;
  gap: 5px;
}

.code-dots span {
  width: 8px;
  height: 8px;
  border-radius: 50%;
}

.code-dots .r { background: #FF5F57; }
.code-dots .y { background: #FFBD2E; }
.code-dots .g { background: #28CA42; }

.code-filename {
  font-size: 11px;
  color: rgba(255, 255, 255, 0.35);
  margin-left: 4px;
}

/* Code body */
.code-body {
  margin: 0;
  padding: 16px;
  color: #c8cce8;
  overflow-x: auto;
  white-space: pre;
}

.code-body code {
  font-family: inherit;
  font-size: inherit;
  color: inherit;
  background: transparent;
  padding: 0;
  border: none;
}

.key { color: #FF2D78; }
.str { color: #FFD60A; }
.op  { color: var(--vp-c-brand-1); }
.cm  { color: rgba(200, 204, 232, 0.3); font-style: italic; }

/* Status bar */
.code-statusbar {
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 6px 14px;
  background: rgba(0, 229, 255, 0.04);
  border-top: 1px solid rgba(255, 255, 255, 0.06);
  font-size: 10px;
  color: rgba(255, 255, 255, 0.3);
}

.status-dot {
  width: 5px;
  height: 5px;
  border-radius: 50%;
  background: #28CA42;
  animation: blink 1.5s ease-in-out infinite;
}

@keyframes blink {
  0%, 100% { opacity: 1; }
  50%       { opacity: 0.3; }
}

@media (max-width: 960px) {
  .hero-code-window {
    max-width: 100%;
    font-size: 12px;
  }
}

@media (max-width: 640px) {
  .hero-code-window {
    font-size: 11px;
  }
}
</style>
