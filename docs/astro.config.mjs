// @ts-check
import { defineConfig } from "astro/config";
import starlight from "@astrojs/starlight";
import starlightThemeRapide from "starlight-theme-rapide";
import starlightSidebarTopics from "starlight-sidebar-topics";
import react from "@astrojs/react";

import mermaid from "astro-mermaid";
import { fileURLToPath } from "url";
import path, { dirname } from "path";

const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);

// https://astro.build/config
export default defineConfig({
  // Deploy to the new v2 subdomain
  site: "https://docs.jan.ai/server",
  integrations: [
    react(),
    mermaid({
      theme: "default",
      autoTheme: true,
    }),
    starlight({
      title: "👋 Jan",
      favicon: "favicon.ico",
      customCss: ["./src/styles/global.css"],
      head: [
        {
          tag: "script",
          attrs: { src: "/scripts/inject-navigation.js", defer: true },
        },
        {
          tag: "link",
          attrs: { rel: "stylesheet", href: "/styles/navigation.css" },
        },
      ],
      plugins: [
        starlightThemeRapide(),
        starlightSidebarTopics(
          [
            {
              label: "Jan",
              link: "https://docs.jan.ai",
              icon: "rocket",
            },
            {
              label: "Jan Desktop",
              link: "https://docs.jan.ai/jan/quickstart",
              icon: "rocket",
            },
            {
              label: "Browser Extension",
              link: "https://docs.jan.ai/browser",
              badge: { text: "Alpha", variant: "tip" },
              icon: "puzzle",
            },
            {
              label: "Jan Mobile",
              link: "/mobile/",
              badge: { text: "Soon", variant: "caution" },
              icon: "phone",
              items: [{ label: "Overview", slug: "mobile" }],
            },
            {
              label: "Jan Server",
              link: "/server/",
              badge: { text: "Soon", variant: "caution" },
              icon: "forward-slash",
              items: [
                { label: "Overview", slug: "server" },
                { label: "Installation", slug: "server/installation" },
                { label: "Architecture", slug: "server/architecture" },
                { label: "Configuration", slug: "server/configuration" },
                { label: "API Reference", slug: "server/api-reference" },
                { label: "Development", slug: "server/development" },
              ],
            },
          ],
          {
            exclude: ["/api-reference", "/api-reference/**/*"],
          },
        ),
      ],
      social: [
        {
          icon: "github",
          label: "GitHub",
          href: "https://github.com/menloresearch/jan",
        },
        {
          icon: "x.com",
          label: "X",
          href: "https://twitter.com/jandotai",
        },
        {
          icon: "discord",
          label: "Discord",
          href: "https://discord.com/invite/FTk2MvZwJH",
        },
      ],
    }),
  ],
});
