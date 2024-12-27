import { defineConfig } from "vitepress";

export default defineConfig({
  title: "kdeps",
  description: "AI agent framework",
  themeConfig: {
    outline: "deep",
    search: {
      provider: "local",
    },
    nav: [{ text: "Home", link: "/" }],
    sidebar: [
      {
        collapsed: false,
        items: [
          {
            text: "Introduction",
            link: "/",
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
            text: "Configuration",
            link: "/getting-started/configuration/configuration",
            items: [
              {
                text: "Workflow",
                link: "/getting-started/configuration/workflow",
              },
              {
                text: "Resources",
                link: "/getting-started/resources/resources",
                items: [
                  {
                    text: "Graph Dependency",
                    link: "/getting-started/resources/kartographer",
                  },
                  {
                    text: "Resource Functions",
                    link: "/getting-started/resources/functions",
                  },
                  {
                    text: "Promise Operator",
                    link: "/getting-started/resources/promise",
                  },
                  {
                    text: "Skip Condition",
                    link: "/getting-started/resources/skipCondition",
                  },
                  {
                    text: "Validations",
                    link: "/getting-started/resources/validations",
                  },
                  {
                    text: "Data folder",
                    link: "/getting-started/resources/data",
                  },
                ],
              },
              {
                text: "Resource Types",
                link: "/getting-started/resources/types",
                items: [
                  {
                    text: "Exec Resource",
                  },
                  {
                    text: "Python Resource",
                  },
                  {
                    text: "HTTP Client Resource",
                  },
                  {
                    text: "LLM Resource",
                  },
                  {
                    text: "API Response Resource",
                  },
                ],
              },
            ],
          },
          {
            text: "Working with files",
          },
          {
            text: "Single-execution mode",
          },
          {
            text: "Security",
          },
          {
            text: "Tutorials",
            items: [
              {
                text: "How to structured LLM resource",
              },
              {
                text: "How to cascade multiple LLM models",
              },
              {
                text: "How to create an AI enhanced OCR",
              },
              {
                text: "How to reuse and extend an AI agent",
              },
              {
                text: "How to use Anaconda in your AI agent",
              },
              {
                text: "How to do image generation",
              },
              {
                text: "How to create a recipe generator",
              },
              {
                text: "How to create an automated JIRA filer",
              },
              {
                text: "How to create an automated TODO creator",
              },
              {
                text: "How to use Huggingface models",
              },
            ],
          },
          {
            text: "Maintenance",
            items: [
              {
                text: "Cleaning Docker Cache",
              },
              {
                text: "Clearing the AI Agents folder",
              },
            ],
          },
          {
            text: "Cloud Services",
            items: [
              {
                text: "Selling your AI agent in the Marketplace",
              },
              {
                text: "Managing your AI agents",
              },
              {
                text: "Developing AI agents in the Kdeps Cloud",
              },
              {
                text: "Hosting and Deploying your AI Agents",
              },
            ],
          },
        ],
      },
    ],

    socialLinks: [{ icon: "github", link: "https://github.com/kdeps/kdeps" }],
  },
});
