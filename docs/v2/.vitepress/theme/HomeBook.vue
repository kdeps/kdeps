<!--
  Copyright 2026 Kdeps, KvK 94834768
  Licensed under the Apache License, Version 2.0
-->
<template>
  <section class="book">
    <div class="container">
      <p class="section-eyebrow">book</p>
      <h2 class="section-title">Go deeper with the official guide</h2>
      <p class="section-sub">Everything from first agent to production deployment, in one place.</p>

      <a href="https://leanpub.com/kdeps" target="_blank" rel="noopener" class="book-card">
        <div class="cover-wrap">
          <img
            src="https://d2sofvawe08yqg.cloudfront.net/kdeps/s_hero?1779817160"
            alt="AI Appliances book cover"
            class="cover"
            loading="lazy"
          />
        </div>
        <div class="book-body">
          <div class="badge-row">
            <span class="badge free">Free</span>
            <span class="badge format">PDF · EPUB · Web</span>
          </div>
          <h3 class="book-title">AI Appliances</h3>
          <p class="book-subtitle">Build &amp; Deploy Autonomous AI Agents and Agencies in YAML</p>
          <p class="book-desc">
            Your AI prototype works. Now ship it. Hands-on guide to deterministic pipelines,
            multi-agent orchestration, error handling, and vendor-agnostic deployment —
            the production challenges most AI frameworks leave to you.
          </p>
          <span class="read-link">Read the book -></span>
        </div>
      </a>
    </div>
  </section>
</template>

<style scoped>
.book {
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

.book-card {
  display: flex;
  gap: 40px;
  align-items: flex-start;
  background: rgba(255, 255, 255, 0.02);
  border: 1px solid rgba(255, 255, 255, 0.07);
  border-radius: 2px;
  padding: 32px;
  text-decoration: none;
  transition: border-color 0.15s ease, transform 0.15s ease;
}

.book-card:hover {
  border-color: rgba(0, 229, 255, 0.25);
  transform: translateY(-2px);
}

.cover-wrap {
  flex-shrink: 0;
  width: 160px;
}

.cover {
  width: 100%;
  display: block;
  border-radius: 2px;
  border: 1px solid rgba(255, 255, 255, 0.08);
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.4);
}

.book-body {
  display: flex;
  flex-direction: column;
  gap: 12px;
  flex: 1;
  min-width: 0;
}

.badge-row {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
}

.badge {
  font-family: var(--vp-font-family-mono);
  font-size: 10px;
  font-weight: 600;
  letter-spacing: 0.08em;
  text-transform: uppercase;
  border-radius: 2px;
  padding: 2px 8px;
}

.free {
  color: #00E5FF;
  background: rgba(0, 229, 255, 0.08);
  border: 1px solid rgba(0, 229, 255, 0.25);
}

.format {
  color: var(--vp-c-text-3);
  background: rgba(255, 255, 255, 0.04);
  border: 1px solid rgba(255, 255, 255, 0.07);
}

.book-title {
  font-size: 24px;
  font-weight: 700;
  letter-spacing: -0.01em;
  color: var(--vp-c-text-1);
  margin: 0;
}

.book-subtitle {
  font-size: 14px;
  font-weight: 500;
  color: var(--vp-c-brand-1);
  margin: 0;
}

.book-desc {
  font-size: 14px;
  line-height: 1.7;
  color: var(--vp-c-text-2);
  margin: 0;
}

.read-link {
  font-size: 14px;
  font-weight: 500;
  color: var(--vp-c-brand-1);
  transition: color 0.15s;
  align-self: flex-start;
}

.book-card:hover .read-link {
  color: #33eaff;
}

@media (max-width: 640px) {
  .book { padding: 48px 16px; }
  .book-card { flex-direction: column; gap: 24px; padding: 20px; }
  .cover-wrap { width: 120px; }
  .section-title { font-size: 28px; }
}
</style>
