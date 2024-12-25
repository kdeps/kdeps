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
            ],
          },
        ],
      },
    ],

    socialLinks: [{ icon: "github", link: "https://github.com/kdeps/kdeps" }],
  },
});
