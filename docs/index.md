---
outline: deep
---

# What is Kdeps?

Kdeps is an all-in-one AI framework for building Dockerized full-stack AI applications (FE and BE) that includes
open-source LLM models out-of-the-box.

## Key Features

Kdeps is loaded with features to streamline AI app development:

- 🐳 Build [Dockerized full-stack AI apps](/getting-started/introduction/quickstart.md#quickstart) with [batteries included](/getting-started/configuration/workflow.md#ai-agent-settings).
- 🔌 Create custom [AI APIs](/getting-started/configuration/workflow.md#api-server-settings) that serve [open-source LLMs](/getting-started/configuration/workflow.md#llm-models).
- 🌐 Pair APIs with [frontend apps](/getting-started/configuration/workflow.md#web-server-settings) like Streamlit, NodeJS, and more.
- 📁 Serve [static websites](/getting-started/configuration/workflow.md#static-file-serving) or [reverse-proxied apps](/getting-started/configuration/workflow.md#reverse-proxying).
- 🔒 Configure [CORS rules](/getting-started/configuration/workflow.md#cors-configuration) directly in the workflow.
- 🛡️ Set [trusted proxies](/getting-started/configuration/workflow.md#trustedproxies) for enhanced API and frontend security.
- 🚀 Run in [Lambda mode](/getting-started/configuration/workflow.md#lambda-mode) or [API mode](/getting-started/configuration/workflow.md#api-server-settings).
- 🤖 Leverage multiple open-source LLMs from [Ollama](/getting-started/configuration/workflow.md#llm-models) and [Huggingface](https://github.com/kdeps/examples/tree/main/huggingface_imagegen_api).
- 🐍 Execute Python in isolated environments using [Anaconda](/getting-started/resources/python.md).
- 🖼️ Support for [multimodal LLMs](/getting-started/resources/multimodal.md).
- ✅ Built-in [API request validations](/getting-started/resources/api-request-validations.md#api-request-validations), [custom validation checks](/getting-started/resources/validations.md), and [skip conditions](/getting-started/resources/skip.md).
- 🔄 Use [reusable AI agents](/getting-started/resources/remix.md) for flexible workflows.
- 🖥️ Run [shell scripts](/getting-started/resources/exec.md) seamlessly.
- 🌍 Make [API calls](/getting-started/resources/client.md) directly from configuration.
- 💾 Manage state with [memory operations](/getting-started/resources/memory.md) to store, retrieve, and clear persistent data.
- 📊 Generate [structured outputs](/getting-started/resources/llm.md#chat-block) from LLMs.
- 📦 Install [Ubuntu packages](/getting-started/configuration/workflow.md#ubuntu-packages) via configuration.
- 📜 Define [Ubuntu repositories or PPAs](/getting-started/configuration/workflow.md#ubuntu-repositories).
- 📈 Enable context-aware [RAG workflows](/getting-started/resources/kartographer.md).
- 🗂️ Upload [documents or files](/getting-started/tutorials/files.md) for LLM processing.
- ⚡ Written in high-performance Golang.
- 📥 [Easy to install](/getting-started/introduction/installation.md) and use with a single command.

## Getting Started

Ready to explore Kdeps? Install it with a single command: [Installation Guide](/getting-started/introduction/installation.md).

Check out practical [examples](https://github.com/kdeps/examples) to jumpstart your projects.

<script setup>
import { withBase } from 'vitepress'
import { useSidebar } from 'vitepress/theme'

const { sidebarGroups } = useSidebar()
</script>
