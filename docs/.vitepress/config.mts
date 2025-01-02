import { defineConfig } from "vitepress";

export default defineConfig({
  title: "kdeps",
  description: "A robust framework for building AI agents",
  themeConfig: {
    outline: "deep",
    search: {
      provider: "local",
    },
    nav: [{ text: "Home", link: "/" }],
    sidebar: [
      {
        text: "Introduction",
        collapsed: false,
        items: [
          {
            text: "Installation",
            link: "/getting-started/introduction/installation",
          },
          {
            text: "Quickstart",
            link: "/getting-started/introduction/quickstart",
          },
        ],
      },
      {
        text: "Configurations",
        items: [
          {
            text: "System-wide Configurations",
            link: "/getting-started/configuration/configuration",
          },
          {
            text: "Workflow",
            link: "/getting-started/configuration/workflow",
            items: [
              {
                text: "API Server Settings",
                link: "/getting-started/configuration/workflow#api-server-settings",
                items: [
                  {
                    text: "API Routes",
                    link: "/getting-started/configuration/workflow#api-routes",
                  },
                ],
              },
              {
                text: "Lambda Mode",
                link: "/getting-started/configuration/workflow#lambda-mode",
              },
              {
                text: "AI Agent Settings",
                link: "/getting-started/configuration/workflow#ai-agent-settings",
                items: [
                  {
                    text: "Anaconda Packages",
                    link: "/getting-started/configuration/workflow#anaconda-packages",
                  },
                  {
                    text: "Python Packages",
                    link: "/getting-started/configuration/workflow#python-packages",
                  },
                  {
                    text: "Ubuntu Repositories",
                    link: "/getting-started/configuration/workflow#ubuntu-repositories",
                  },
                  {
                    text: "Ubuntu Packages",
                    link: "/getting-started/configuration/workflow#ubuntu-packages",
                  },
                  {
                    text: "LLM Models",
                    link: "/getting-started/configuration/workflow#llm-models",
                  },
                  {
                    text: "Ollama Docker Image Tag",
                    link: "/getting-started/configuration/workflow#ollama-docker-image-tag",
                  },
                  {
                    text: "Arguments and Environment Variables",
                    link: "/getting-started/configuration/workflow#arguments-and-environment-variables",
                  },
                ],
              },
            ],
          },
        ],
      },
      {
        text: "Resources",
        items: [
          {
            text: "Graph Dependency",
            link: "/getting-started/resources/kartographer",
          },
          {
            text: "Functions",
            link: "/getting-started/resources/functions",
            items: [
              {
                text: "Exec",
                link: "/getting-started/resources/functions#exec-resource-functions",
              },
              {
                text: "HTTP Client",
                link: "/getting-started/resources/functions#http-client-resource-functions",
              },
              {
                text: "LLM",
                link: "/getting-started/resources/functions#llm-resource-functions",
              },
              {
                text: "Python",
                link: "/getting-started/resources/functions#python-resource-functions",
              },
              {
                text: "API Request",
                link: "/getting-started/resources/functions#api-request-functions",
              },
            ],
          },
          {
            text: "Promise Operator",
            link: "/getting-started/resources/promise",
          },
          { text: "Skip Conditions", link: "/getting-started/resources/skip" },
          {
            text: "Preflight Validations",
            link: "/getting-started/resources/validations",
          },
          { text: "Data Folder", link: "/getting-started/resources/data" },
          { text: "File Uploads", link: "/getting-started/tutorials/files" },
        ],
      },
      {
        text: "Resource Types",
        items: [
          { text: "Exec Resource", link: "/getting-started/resources/exec" },
          {
            text: "Python Resource",
            link: "/getting-started/resources/python",
          },
          {
            text: "HTTP Client Resource",
            link: "/getting-started/resources/client",
          },
          { text: "LLM Resource", link: "/getting-started/resources/llm" },
          {
            text: "API Response Resource",
            link: "/getting-started/resources/response",
          },
        ],
      },
      {
        text: "Tutorials",
        items: [
          {
            text: "How to create structured LLM response APIs",
            link: "/getting-started/tutorials/how-to-structure-llm",
          },
          { text: "How to cascade multiple LLM models" },
          { text: "How to create an AI-enhanced OCR" },
          { text: "How to reuse and extend an AI agent" },
          { text: "How to use Anaconda in your AI agent" },
          { text: "How to do image generation" },
          { text: "How to create a recipe generator" },
          { text: "How to create an automated JIRA filer" },
          { text: "How to create an automated TODO creator" },
          { text: "How to use Huggingface models" },
        ],
      },
      {
        text: "Maintenance",
        items: [
          { text: "Cleaning Docker Cache" },
          { text: "Clearing the AI Agents Folder" },
        ],
      },
      {
        text: "Cloud Services",
        items: [
          { text: "Selling your AI agent in the Marketplace" },
          { text: "Managing your AI agents" },
          { text: "Developing AI agents in the Kdeps Cloud" },
          { text: "Hosting and Deploying your AI Agents" },
        ],
      },
    ],
    socialLinks: [{ icon: "github", link: "https://github.com/kdeps/kdeps" }],
  },
});
