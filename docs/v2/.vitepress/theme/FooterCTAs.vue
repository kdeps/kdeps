<!--
  Copyright 2026 Kdeps, KvK 94834768
  Licensed under the Apache License, Version 2.0
-->
<script setup>
import { useData } from 'vitepress'
const { frontmatter } = useData()
</script>

<template>
  <div v-if="frontmatter.layout === 'home'" class="footer-ctas">
    <a class="primary" href="/getting-started/installation">Get Started</a>
    <a class="secondary" href="https://github.com/kdeps/kdeps" target="_blank">Star on GitHub</a>
  </div>
</template>
