# What is Kdeps?

Kdeps is a no-code framework for building self-hosted RAG AI Agents powered by open-source LLMs.

And...
1. It uses open-source LLMs by default.
2. Has a built-in context-aware RAG workflow system.
3. Builds a Docker image of the AI Agent.

If Ollama, RAG, API and Docker had a lovechild, it would be Kdeps.

<img alt="Kdeps - Overview" src="/docs/public/overview.png" />

Kdeps is packed with features:
- 🚀 run in [Lambda](https://kdeps.github.io/kdeps/getting-started/configuration/workflow.html#lambda-mode) or [API Mode](https://kdeps.github.io/kdeps/getting-started/configuration/workflow.html#api-server-settings)
- 🤖 use multiple open-source LLMs from [Ollama](https://kdeps.github.io/kdeps/getting-started/configuration/workflow.html#llm-models) and [Huggingface](https://github.com/kdeps/examples/tree/main/huggingface_imagegen_api)
- 🐍 run Python in isolated environments using [Anaconda](https://kdeps.github.io/kdeps/getting-started/resources/python.html)
- 🖼️ [multimodal](https://kdeps.github.io/kdeps/getting-started/resources/multimodal.html) LLMs ready
- 💅 built-in [validation](https://kdeps.github.io/kdeps/getting-started/resources/validations.html) checks and [skip](https://kdeps.github.io/kdeps/getting-started/resources/skip.html) conditions
- 🔄 [reusable](https://kdeps.github.io/kdeps/getting-started/resources/remix.html) AI Agents
- 🖥️ run [shell-scripts](https://kdeps.github.io/kdeps/getting-started/resources/exec.html)
- 🌐 make [API calls](https://kdeps.github.io/kdeps/getting-started/resources/client.html) from configuration
- 📊 generate [structured outputs](https://kdeps.github.io/kdeps/getting-started/resources/llm.html#chat-block) from LLMs
- 📦 install [Ubuntu packages](https://kdeps.github.io/kdeps/getting-started/configuration/workflow.html#ubuntu-packages) from configuration
- 📜 define [Ubuntu repos or PPAs](https://kdeps.github.io/kdeps/getting-started/configuration/workflow.html#ubuntu-repositories)
- 📈 context-aware [RAG workflow](https://kdeps.github.io/kdeps/getting-started/resources/kartographer.html)
- 🗂️ upload any [documents or files](https://kdeps.github.io/kdeps/getting-started/tutorials/files.html) for LLM processing
- ⚡ Written in Golang
- 📦 [easy to install](https://kdeps.github.io/kdeps/getting-started/introduction/installation.html) and use

I know, that's a lot. Let's dive into the details.

You can get started with Kdeps [via installing it](https://kdeps.github.io/kdeps/getting-started/introduction/installation.html) with a single command.

See the [examples](https://github.com/kdeps/examples).
