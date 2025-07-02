import { defineConfig } from "vitepress";

export default defineConfig({
  title: "kdeps",
  description: "A robust framework for building AI agents with v0.3.1 schema",
  themeConfig: {
    outline: "deep",
    search: {
      provider: "local",
    },
    nav: [{ text: "Home", link: "/" }],
    sidebar: [
      {
        text: "🚀 Getting Started",
        collapsed: false,
        items: [
          {
            text: "Installation",
            link: "/getting-started/introduction/installation",
          },
          {
            text: "Quickstart Guide",
            link: "/getting-started/introduction/quickstart",
          },
        ],
      },
      {
        text: "⚙️ Configuration",
        collapsed: false,
        items: [
          {
            text: "System Configuration",
            link: "/getting-started/configuration/configuration",
          },
          {
            text: "Workflow Configuration",
            link: "/getting-started/configuration/workflow",
            items: [
              {
                text: "API Server Settings",
                link: "/getting-started/configuration/workflow#api-server-settings",
                items: [
                  {
                    text: "Trusted Proxies",
                    link: "/getting-started/configuration/workflow#trustedproxies",
                  },
                  {
                    text: "CORS Configuration",
                    link: "/getting-started/configuration/workflow#cors-configuration",
                  },
                  {
                    text: "API Routes",
                    link: "/getting-started/configuration/workflow#api-routes",
                  },
                ],
              },
              {
                text: "Web Server Settings",
                link: "/getting-started/configuration/workflow#web-server-settings",
                items: [
                  {
                    text: "Web Server",
                    link: "/getting-started/configuration/workflow#webserver",
                  },
                  {
                    text: "Web Server Routes",
                    link: "/getting-started/configuration/workflow#web-server-routes",
                    items: [
                      {
                        text: "Static File Serving",
                        link: "/getting-started/configuration/workflow#static-file-serving",
                      },
                      {
                        text: "Reverse Proxying",
                        link: "/getting-started/configuration/workflow#reverse-proxying",
                      },
                    ],
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
                    text: "LLM Models",
                    link: "/getting-started/configuration/workflow#llm-models",
                  },
                  {
                    text: "Ollama Version",
                    link: "/getting-started/configuration/workflow#ollama-docker-image-tag",
                  },
                  {
                    text: "Python Packages",
                    link: "/getting-started/configuration/workflow#python-packages",
                  },
                  {
                    text: "Anaconda Packages",
                    link: "/getting-started/configuration/workflow#anaconda-packages",
                  },
                  {
                    text: "Ubuntu Packages",
                    link: "/getting-started/configuration/workflow#ubuntu-packages",
                  },
                  {
                    text: "Environment Variables",
                    link: "/getting-started/configuration/workflow#arguments-and-environment-variables",
                  },
                ],
              },
            ],
          },
          {
            text: "CORS Configuration",
            link: "/getting-started/configuration/cors",
          },
          {
            text: "Web Server Configuration",
            link: "/getting-started/configuration/webserver",
          },
        ],
      },
      {
        text: "🔧 Core Resources",
        collapsed: false,
        items: [
          {
            text: "Resources Overview",
            link: "/resources",
          },
          {
            text: "LLM Resource",
            link: "/core-resources/llm",
          },
          {
            text: "API Response Resource",
            link: "/core-resources/response",
          },
          {
            text: "HTTP Client Resource",
            link: "/core-resources/client",
          },
          {
            text: "Python Resource",
            link: "/core-resources/python",
          },
          {
            text: "Exec Resource",
            link: "/core-resources/exec",
          },
        ],
      },
      {
        text: "🛠️ Advanced Resources",
        collapsed: false,
        items: [
          {
            text: "Multi-Modal LLM Models",
            link: "/advanced-resources/multimodal",
          },
          {
            text: "AI Image Generators",
            link: "/advanced-resources/image-generators",
          },
          {
            text: "Tool Calling (MCP)",
            link: "/advanced-resources/tools",
          },
          {
            text: "Items Iteration",
            link: "/advanced-resources/items",
          },
        ],
      },
      {
        text: "🔗 Workflow Control",
        collapsed: false,
        items: [
          {
            text: "Graph Dependency",
            link: "/workflow-control/kartographer",
          },
          {
            text: "Skip Conditions",
            link: "/workflow-control/skip",
          },
          {
            text: "Preflight Validations",
            link: "/workflow-control/validations",
          },
          {
            text: "API Request Validations",
            link: "/workflow-control/api-request-validations",
          },
          {
            text: "Promise Operator",
            link: "/workflow-control/promise",
          },
        ],
      },
      {
        text: "💾 Data & Memory",
        collapsed: false,
        items: [
          {
            text: "Memory Operations",
            link: "/data-memory/memory",
          },
          {
            text: "Data Folder",
            link: "/data-memory/data",
          },
          {
            text: "Working with JSON",
            link: "/data-memory/json",
          },
          {
            text: "File Uploads",
            link: "/data-memory/files",
          },
        ],
      },
      {
        text: "⚡ Functions & Utilities",
        collapsed: false,
        items: [
          {
            text: "Resource Functions",
            link: "/functions-utilities/functions",
            items: [
              {
                text: "LLM Functions",
                link: "/functions-utilities/functions#llm-resource-functions",
              },
              {
                text: "HTTP Client Functions",
                link: "/functions-utilities/functions#http-client-resource-functions",
              },
              {
                text: "Python Functions",
                link: "/functions-utilities/functions#python-resource-functions",
              },
              {
                text: "Exec Functions",
                link: "/functions-utilities/functions#exec-resource-functions",
              },
            ],
          },
          {
            text: "Global Functions",
            link: "/functions-utilities/global-functions",
            items: [
              {
                text: "API Request Functions",
                link: "/functions-utilities/global-functions#api-request-functions",
              },
              {
                text: "Data Folder Functions",
                link: "/functions-utilities/global-functions#data-folder-functions",
              },
              {
                text: "Memory Operations",
                link: "/functions-utilities/global-functions#memory-operation-functions",
              },
              {
                text: "JSON Document Parser",
                link: "/functions-utilities/global-functions#document-json-parsers",
              },
              {
                text: "Document Generators",
                link: "/functions-utilities/global-functions#document-json-yaml-and-xml-generators",
              },
              {
                text: "Skip Condition Helpers",
                link: "/functions-utilities/global-functions#skip-condition-functions",
              },
              {
                text: "PKL Modules",
                link: "/functions-utilities/global-functions#pkl-modules",
              },
            ],
          },
          {
            text: "Expr Block",
            link: "/functions-utilities/expr",
          },
          {
            text: "Data Types",
            link: "/functions-utilities/types",
          },
        ],
      },
      {
        text: "🔄 Reusability",
        collapsed: false,
        items: [
          {
            text: "Reusing and Remixing AI Agents",
            link: "/reusability/remix",
          },
        ],
      },
      {
        text: "📚 Tutorials",
        collapsed: false,
        items: [
          {
            text: "Weather API Tutorial",
            link: "/tutorials/how-to-weather-api",
          },
          {
            text: "Structured LLM Responses",
            link: "/tutorials/how-to-structure-llm",
          },
        ],
      },
      {
        text: "☁️ Cloud Services",
        collapsed: false,
        items: [
          {
            text: "AI Agent Marketplace",
            link: "#",
          },
          {
            text: "Agent Management",
            link: "#",
          },
          {
            text: "Cloud Development",
            link: "#",
          },
          {
            text: "Hosting & Deployment",
            link: "#",
          },
        ],
      },
    ],
    socialLinks: [{ icon: "github", link: "https://github.com/kdeps/kdeps" }],
  },
});
