// @ts-check
import { defineConfig } from 'astro/config';
import starlight from '@astrojs/starlight';
import { unified } from '@astrojs/markdown-remark';
import rehypeBaseLinks from './src/plugins/rehype-base-links.mjs';
import rehypeFocusableTables from './src/plugins/rehype-focusable-tables.mjs';

// The site is a GitHub Pages project site, served under /quiver/. For a
// custom domain, build with SITE_URL=https://example.dev BASE_PATH=/ and add
// public/CNAME (see README.md).
const site = process.env.SITE_URL ?? 'https://kheif.github.io';
const base = process.env.BASE_PATH ?? '/quiver';

const repo = 'https://github.com/kheif/quiver';

export default defineConfig({
	site,
	base,
	markdown: {
		processor: unified({
			// Keep -- in flags such as --global; no typographic dashes.
			smartypants: false,
			// Docs link to other pages as /docs/...; this adds the base path.
			rehypePlugins: [[rehypeBaseLinks, { base }], rehypeFocusableTables],
		}),
	},
	integrations: [
		starlight({
			title: 'Quiver',
			description:
				'Quiver keeps your AI agent skills in one library and serves the right ones to each project over MCP.',
			logo: { dark: './src/assets/mark-dark.svg', light: './src/assets/mark-light.svg', alt: '' },
			favicon: '/favicon.svg',
			social: [{ icon: 'github', label: 'GitHub', href: repo }],
			components: { SocialIcons: './src/components/DocsSocial.astro' },
			editLink: { baseUrl: `${repo}/edit/main/web/` },
			customCss: [
				'@fontsource-variable/hanken-grotesk',
				'@fontsource-variable/jetbrains-mono',
				'./src/styles/tokens.css',
				'./src/styles/starlight.css',
			],
			head: [
				{ tag: 'meta', attrs: { property: 'og:image', content: new URL(`${base.replace(/\/$/, '')}/og.png`, site).href } },
				{ tag: 'meta', attrs: { name: 'twitter:card', content: 'summary_large_image' } },
			],
			expressiveCode: {
				themes: ['github-dark-default', 'github-light-default'],
				styleOverrides: {
					borderRadius: '0.5rem',
					codeFontFamily: 'var(--q-font-mono)',
					uiFontFamily: 'var(--q-font-sans)',
				},
			},
			sidebar: [
				{
					label: 'Getting started',
					items: [
						'docs/introduction',
						'docs/installation',
						'docs/quickstart',
						'docs/assistants',
					],
				},
				{
					label: 'Concepts',
					items: [
						'docs/concepts/library',
						'docs/concepts/layers',
						'docs/concepts/assignments',
						'docs/concepts/always-on',
						'docs/concepts/projects',
						'docs/concepts/mcp-delivery',
					],
				},
				{
					label: 'Guides',
					items: [
						'docs/guides/writing-skills',
						'docs/guides/importing-skills',
						'docs/guides/sources',
						'docs/guides/updating-and-pinning',
						'docs/guides/multiple-machines',
						'docs/guides/troubleshooting',
					],
				},
				{
					label: 'Reference',
					items: [
						'docs/reference/cli',
						'docs/reference/mcp',
						'docs/reference/files',
						'docs/reference/environment',
						'docs/reference/platforms',
					],
				},
				{
					label: 'Project',
					items: [
						'docs/project/architecture',
						'docs/project/security',
						'docs/project/contributing',
						'docs/project/releases',
					],
				},
			],
		}),
	],
});
