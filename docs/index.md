---
outline: deep
---

# What is Kdeps?

Kdeps is a no-code framework for building self-hosted RAG AI Agents powered by open-source LLMs.

1. It uses open-source LLMs by default.
2. Has a built-in context-aware RAG workflow system.
3. Builds a Docker image of the AI Agent.

<img alt="Kdeps - Overview" src="/overview.png" />

Kdeps is packed with features:
- 🚀 run in [Lambda](getting-started/configuration/workflow.md#lambda-mode) or [API Mode](getting-started/configuration/workflow.md#api-server-settings)
- 🤖 use multiple open-source LLMs from [Ollama](getting-started/configuration/workflow.md#llm-models) and [Huggingface](https://github.com/kdeps/examples/tree/main/huggingface_imagegen_api)
- 🐍 run Python in isolated environments using [Anaconda](getting-started/resources/python.md)
- 🖼️ [multimodal](getting-started/resources/multimodal.md) LLMs ready
- 💅 built-in [validation](getting-started/resources/validations.md) checks and [skip](getting-started/resources/skip.md) conditions
- 🔄 [reusable](getting-started/resources/remix.md) AI Agents
- 🖥️ run [shell-scripts](getting-started/resources/exec.md)
- 🌐 make [API calls](getting-started/resources/client.md) from configuration
- 📊 generate [structured outputs](getting-started/resources/llm.md#chat-block) from LLMs
- 📦 install [Ubuntu packages](getting-started/configuration/workflow.md#ubuntu-packages) from configuration
- 📜 define [Ubuntu repos or PPAs](getting-started/configuration/workflow.md#ubuntu-repositories)
- 📈 context-aware [RAG workflow](getting-started/resources/kartographer.md)
- 🗂️ upload any [documents or files](getting-started/tutorials/files.md) for LLM processing
- ⚡ Written in Golang
- 📦 [easy to install](getting-started/introduction/installation.md) and use

I know, that's a lot. Let's dive into the details.

You can get started with Kdeps [via installing it](getting-started/introduction/installation.md) with a single command.

See the [examples](https://github.com/kdeps/examples).

<script setup>
import { withBase } from 'vitepress'
import { useSidebar } from 'vitepress/theme'

const { sidebarGroups } = useSidebar()
</script>
