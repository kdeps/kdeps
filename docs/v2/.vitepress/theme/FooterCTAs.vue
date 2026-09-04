<!--
  Copyright 2026 Kdeps, KvK 94834768
  Licensed under the Apache License, Version 2.0
-->
<script setup>
import { useData } from 'vitepress'
const { frontmatter } = useData()
</script>

<template>
  <div v-if="frontmatter.layout === 'home'">
    <div class="footer-ctas">
      <a class="primary" href="/getting-started/local-agent">Get Started</a>
      <a class="secondary" href="https://github.com/kdeps/kdeps" target="_blank">Star on GitHub</a>
    </div>
    <p class="footer-note">Proud member of the NVIDIA Inception program.</p>
  </div>
</template>

<style scoped>
.footer-note {
  text-align: center;
  font-size: 12px;
  color: var(--vp-c-text-3);
  margin: 0 0 40px;
}
</style>
