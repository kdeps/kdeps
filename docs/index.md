---
# https://vitepress.dev/reference/default-theme-home-page
layout: home

hero:
  name: "Kdeps"
  text: "AI Agent Framework"
  tagline: |
    Kdeps is a framework that reduces the complexity of building self-hosted APIs powered by open-source LLMs.

    It packs everything an RAG AI Agent needs in a Dockerized image, that can be deployed on/off cloud anytime, anywhere.

    It uses open-source LLMs, so there's no need to buy any AI services and subscriptions for your AI-powered APIs.

  actions:
    - theme: brand
      text: Installation
      link: /getting-started/introduction/installation
    - theme: alt
      text: Quickstart
      link: /getting-started/introduction/quickstart
    - theme: alt
      text: Examples
      link: https://github.com/kdeps/examples
    - theme: alt
      text: Github
      link: https://github.com/kdeps/kdeps

  image:
    src: /demo.gif
    alt: Kdeps


features:
  - title: 💡 Kdeps is easy, practical, and no-code
  - title: 🚀 Run Kdeps in Lambda or API Mode
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
  - title: 🌐 OPENAPI and JSONAPI Compatible
  - title: 🆓 Use free and open-source LLMs with no subscriptions
  - title: 🖼️ Use Multimodal LLMs
  - title: 🧠 Develop intelligent and context-aware APIs
  - title: 🎨 Create AI image generator APIs
  - title: 🗂️ Upload any documents or files for LLM processing
  - title: ⚡ Written in Golang
---

<script setup>
import DefaultTheme from 'vitepress/theme';
import '/public/custom.css';
</script>
