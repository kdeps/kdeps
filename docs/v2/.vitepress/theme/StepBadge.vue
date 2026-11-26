<!--
  Copyright 2026 Kdeps, KvK 94834768
  Licensed under the Apache License, Version 2.0
-->
<script setup>
import { computed } from 'vue'
import { useData } from 'vitepress'

const { frontmatter } = useData()
const step = computed(() => frontmatter.value.step)
const pct = computed(() => (step.value ? Math.round((step.value.n / step.value.total) * 100) : 0))
</script>

<template>
  <div v-if="step" class="step-badge">
    <p class="label">step {{ step.n }} of {{ step.total }} &middot; {{ step.part }}</p>
    <div class="bar" role="progressbar" :aria-valuenow="step.n" aria-valuemin="1" :aria-valuemax="step.total">
      <i :style="{ width: pct + '%' }"></i>
    </div>
  </div>
</template>

<style scoped>
.step-badge {
  margin: 0 0 24px;
}

.label {
  font-family: var(--vp-font-family-mono);
  font-size: 11px;
  text-transform: uppercase;
  letter-spacing: 0.08em;
  color: var(--vp-c-brand-1);
  margin: 0 0 8px;
}

.bar {
  height: 2px;
  background: rgba(255, 255, 255, 0.06);
  border-radius: 2px;
  overflow: hidden;
}

.bar i {
  display: block;
  height: 100%;
  background: var(--vp-c-brand-1);
  transition: width 0.15s ease;
}
</style>
