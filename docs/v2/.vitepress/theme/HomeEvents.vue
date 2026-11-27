<!--
  Copyright 2026 Kdeps, KvK 94834768
  Licensed under the Apache License, Version 2.0
-->
<template>
  <section class="events">
    <div class="container">
      <p class="section-eyebrow">events in, calls out</p>
      <h2 class="section-title">A function per event</h2>
      <p class="section-sub">Each event starts one bounded run, as with a serverless function. A listener receives the event, and a connector sends the answer. Keys stay in an encrypted vault.</p>

      <div class="kd-play flow">
        <a href="/workflow/input-sources" class="kd-stagger col" style="--i: 0">
          <span class="col-label">listen:</span>
          <ul>
            <li>telegram_updates</li>
            <li>slack_events</li>
            <li>discord_gateway</li>
            <li>whatsapp_webhook</li>
            <li class="dim">webhook &middot; websocket &middot; poll &middot; file</li>
          </ul>
        </a>

        <div class="arrow" aria-hidden="true">&rarr;</div>

        <a href="/workflow/" class="kd-stagger col center" style="--i: 1">
          <span class="col-label">workflow.yaml</span>
          <p class="node">one run</p>
          <p class="hint">checks, a model, an agent step</p>
        </a>

        <div class="arrow" aria-hidden="true">&rarr;</div>

        <a href="/workflow/resources/connectors" class="kd-stagger col" style="--i: 2">
          <span class="col-label">connectors</span>
          <ul>
            <li>telegram_post</li>
            <li>slack_post</li>
            <li>github_post</li>
            <li>notion_post</li>
            <li class="dim">discord &middot; whatsapp &middot; jira &middot; stripe &middot; airtable</li>
          </ul>
        </a>
      </div>

      <div class="links">
        <a href="/workflow/input-sources">Listeners -></a>
        <a href="/workflow/resources/connectors">Connectors -></a>
        <a href="/workflow/secrets">Secrets vault -></a>
      </div>
    </div>
  </section>
</template>

<style scoped>
.events {
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

.flow {
  display: flex;
  align-items: stretch;
  gap: 16px;
}

.col {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 12px;
  padding: 20px;
  background: var(--vp-c-bg-soft);
  border: 1px solid rgba(255, 255, 255, 0.07);
  border-radius: 2px;
  text-decoration: none;
  color: inherit;
  transition: transform 0.15s ease, border-color 0.15s ease;
}

.col:hover {
  transform: translateY(-2px);
  border-color: rgba(0, 229, 255, 0.35);
}

.col:focus-visible {
  outline: 2px solid var(--vp-c-brand-1);
  outline-offset: 2px;
}

.col-label {
  font-family: var(--vp-font-family-mono);
  font-size: 11px;
  font-weight: 600;
  letter-spacing: 0.08em;
  color: var(--vp-c-brand-1);
}

.col ul {
  list-style: none;
  margin: 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.col li {
  font-family: var(--vp-font-family-mono);
  font-size: 13px;
  color: var(--vp-c-text-1);
}

.col li.dim,
.hint {
  font-size: 12px;
  color: var(--vp-c-text-3);
}

.center {
  justify-content: center;
  align-items: center;
  text-align: center;
  border-color: rgba(0, 229, 255, 0.25);
}

.node {
  font-size: 17px;
  font-weight: 600;
  color: var(--vp-c-text-1);
  margin: 0;
}

.hint {
  margin: 0;
}

.arrow {
  align-self: center;
  font-family: var(--vp-font-family-mono);
  font-size: 18px;
  color: rgba(255, 255, 255, 0.2);
  flex-shrink: 0;
}

.links {
  display: flex;
  flex-wrap: wrap;
  gap: 24px;
  margin-top: 24px;
}

.links a {
  font-size: 13px;
  color: var(--vp-c-brand-1);
  text-decoration: none;
}

.links a:hover {
  text-decoration: underline;
}

@media (max-width: 768px) {
  .events {
    padding: 48px 24px;
  }

  .flow {
    flex-direction: column;
  }

  .arrow {
    transform: rotate(90deg);
  }
}
</style>
