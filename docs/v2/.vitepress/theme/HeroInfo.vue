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
    <img src="/kdeps-logo.png" alt="kdeps" class="hero-logo" />
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

.hero-logo {
  height: 64px;
  width: auto;
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
