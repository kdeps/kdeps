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
      <span class="code-filename">agent.yaml</span>
    </div>

    <!-- Code body -->
    <div class="code-body">
      <div><span class="ln"> 1</span><span class="key">name</span><span class="op">:</span> <span class="str">support-agent</span></div>
      <div><span class="ln"> 2</span><span class="key">resources</span><span class="op">:</span></div>
      <div><span class="ln"> 3</span>  <span class="op">-</span> <span class="key">type</span><span class="op">:</span> <span class="str">db</span></div>
      <div><span class="ln"> 4</span>    <span class="key">query</span><span class="op">:</span> <span class="str">"SELECT * FROM tickets"</span></div>
      <div><span class="ln"> 5</span>  <span class="op">-</span> <span class="key">type</span><span class="op">:</span> <span class="str">python</span></div>
      <div><span class="ln"> 6</span>    <span class="key">script</span><span class="op">:</span> <span class="str">classify.py</span></div>
      <div><span class="ln"> 7</span>  <span class="op">-</span> <span class="key">type</span><span class="op">:</span> <span class="str">llm</span></div>
      <div><span class="ln"> 8</span>    <span class="key">model</span><span class="op">:</span> <span class="str">gpt-4o-mini</span></div>
      <div><span class="ln"> 9</span>    <span class="key">prompt</span><span class="op">:</span> <span class="str">"Resolve: {{ticket}}"</span></div>
      <div><span class="ln">10</span>    <span class="key">depends_on</span><span class="op">:</span> <span class="str">[db, python]</span></div>
      <div class="blank-line"></div>
      <div><span class="cm"># That's the whole agent.</span></div>
    </div>

    <!-- Status bar -->
    <div class="code-statusbar">
      <span class="status-dot"></span>
      <span>running · port 8080</span>
      <span>startup: 0.4s</span>
    </div>
  </div>
</template>

<style scoped>
.hero-code-window {
  font-family: 'JetBrains Mono', 'Fira Code', monospace;
  font-size: 13px;
  line-height: 1.75;
  background: #070707;
  border: 1px solid rgba(255, 255, 255, 0.08);
  border-left: 2px solid rgba(0, 229, 255, 0.4);
  border-radius: 2px;
  overflow: hidden;
  width: 100%;
  max-width: 380px;
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
  padding: 12px 16px;
  color: #c8cce8;
}

.code-body div {
  white-space: pre;
}

.blank-line {
  height: 0.5em;
}

.ln  { color: rgba(0, 229, 255, 0.18); margin-right: 14px; user-select: none; }
.key { color: #FF2D78; }
.val { color: #28CA42; }
.str { color: #FFD60A; }
.op  { color: #00E5FF; }
.cm  { color: rgba(200, 204, 232, 0.3); font-style: italic; }

/* Status bar */
.code-statusbar {
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 5px 14px;
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
</style>
