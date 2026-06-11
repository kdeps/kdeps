<!--
  Copyright 2026 Kdeps, KvK 94834768
  Licensed under the Apache License, Version 2.0
-->
<script setup lang="ts">
import { useData } from 'vitepress'
const { frontmatter } = useData()
</script>

<template>
  <div class="hero-info">
    <a
      v-if="frontmatter.hero?.announcement && frontmatter.hero?.announcementLink"
      :href="frontmatter.hero.announcementLink"
      class="announcement announcement-link"
    >
      {{ frontmatter.hero.announcement }}
    </a>
    <span v-else-if="frontmatter.hero?.announcement" class="announcement">
      {{ frontmatter.hero.announcement }}
    </span>
    <p v-if="frontmatter.hero?.text" class="hero-text">
      {{ frontmatter.hero.text }}
    </p>
    <p v-if="frontmatter.hero?.tagline" class="hero-tagline">
      {{ frontmatter.hero.tagline }}
    </p>
  </div>
</template>

<style scoped>
.hero-info {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 12px;
}

.announcement {
  font-family: var(--vp-font-family-mono);
  font-size: 11px;
  font-weight: 600;
  letter-spacing: 0.08em;
  text-transform: uppercase;
  color: var(--vp-c-brand-1);
  border: 1px solid rgba(0, 229, 255, 0.3);
  border-radius: 2px;
  padding: 4px 10px;
  margin-bottom: 4px;
}

.announcement-link {
  text-decoration: none;
  transition: border-color 0.2s, background-color 0.2s;
}

.announcement-link:hover {
  border-color: rgba(0, 229, 255, 0.6);
  background-color: rgba(0, 229, 255, 0.08);
}

.hero-text {
  margin: 0;
  font-size: clamp(28px, 5vw, 52px);
  font-weight: 700;
  letter-spacing: -0.02em;
  line-height: 1.15;
  color: var(--vp-c-text-1);
}

.hero-tagline {
  margin: 0;
  max-width: 480px;
  font-size: clamp(15px, 2vw, 20px);
  font-weight: 400;
  line-height: 1.6;
  color: var(--vp-c-text-2);
}
</style>
