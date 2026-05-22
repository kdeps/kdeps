<!--
  Copyright 2026 Kdeps, KvK 94834768
  Licensed under the Apache License, Version 2.0
-->
<template>
  <div class="hero-terminal">
    <div class="terminal-titlebar">
      <div class="code-dots">
        <span class="r"></span>
        <span class="y"></span>
        <span class="g"></span>
      </div>
      <span class="terminal-title">terminal</span>
    </div>

    <div class="terminal-body">
      <div class="line">
        <span class="prompt">$</span>
        <span class="cmd">kdeps run workflow.yaml</span>
      </div>
      <div class="line output">Listening on :16395</div>
      <div class="line spacer"></div>
      <div class="line">
        <span class="prompt">$</span>
        <span class="cmd">curl localhost:16395/ask \</span>
      </div>
      <div class="line">
        <span class="indent"></span>
        <span class="cmd dim">-d '{"q": "explain REST in one line"}'</span>
      </div>
      <div class="line spacer"></div>
      <div class="line response">{"response": "REST is HTTP + URLs + JSON, stateless by design."}</div>
    </div>

    <div class="terminal-statusbar">
      <span class="status-dot"></span>
      <span>running</span>
      <span>port 16395</span>
    </div>
  </div>
</template>

<style scoped>
.hero-terminal {
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

.terminal-titlebar {
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

.terminal-title {
  font-size: 11px;
  color: rgba(255, 255, 255, 0.35);
  margin-left: 4px;
}

.terminal-body {
  padding: 16px;
  color: #c8cce8;
}

.line {
  display: flex;
  gap: 8px;
  white-space: pre;
}

.line.spacer {
  height: 8px;
}

.prompt {
  color: var(--vp-c-brand-1);
  user-select: none;
  flex-shrink: 0;
}

.cmd {
  color: #c8cce8;
}

.cmd.dim {
  color: rgba(200, 204, 232, 0.6);
}

.indent {
  display: inline-block;
  width: 16px;
  flex-shrink: 0;
}

.output {
  color: rgba(200, 204, 232, 0.4);
  padding-left: 16px;
}

.response {
  color: #FFD60A;
  padding-left: 16px;
  white-space: pre-wrap;
  word-break: break-word;
}

.terminal-statusbar {
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
  .hero-terminal {
    max-width: 100%;
    font-size: 12px;
  }
}

@media (max-width: 640px) {
  .hero-terminal {
    font-size: 11px;
  }
}
</style>
