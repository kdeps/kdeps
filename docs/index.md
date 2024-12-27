---
# https://vitepress.dev/reference/default-theme-home-page
layout: home

hero:
  name: "Kdeps"
  text: "AI Agent Framework"
  tagline: Kdeps is a multi-model AI agent framework that is optimized for creating purpose-built Dockerized RAG AI agents APIs ready to be deployed in the cloud.

  actions:
    - theme: brand
      text: Installation
      link: /getting-started/introduction/installation
    - theme: alt
      text: Quickstart
      link: /getting-started/introduction/quickstart
    - theme: alt
      text: Github
      link: https://github.com/kdeps/kdeps

  image:
    src: /demo.gif
    alt: Kdeps


features:
  - title: 🚀 Run Kdeps in single-execution or API Mode
  - title: 🤖 Use Multiple Open-Source LLMs
  - title: 🐍 Run Python scripts in isolated environments using Anaconda
  - title: 🖥️ Execute Custom Shell-Scripts
  - title: 🧪 Anaconda Support
  - title: 🔄 Share and Remix AI Agents
  - title: 🌐 Interact with external HTTP APIs directly into the resource
  - title: 📊 Generate structured outputs from LLMs
  - title: 📦 Install dependent Ubuntu packages from within the workflow configuration
  - title: 📜 Define custom Ubuntu repositories and PPAs in the workflow
  - title: 📈 RAG Graph-based workflow execution
  - title: ⚡ Written in Golang
---

<script setup>
import DefaultTheme from 'vitepress/theme';
import '/public/custom.css';
</script>
