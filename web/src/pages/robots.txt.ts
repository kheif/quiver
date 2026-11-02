import type { APIRoute } from 'astro';

// Crawlers read robots.txt only at the root of a domain, so this takes effect
// once the site has its own domain; under /quiver/ it is harmless.
export const GET: APIRoute = ({ site }) => {
	const sitemap = new URL(`${import.meta.env.BASE_URL.replace(/\/$/, '')}/sitemap-index.xml`, site);
	return new Response(`User-agent: *\nAllow: /\n\nSitemap: ${sitemap.href}\n`, {
		headers: { 'Content-Type': 'text/plain; charset=utf-8' },
	});
};
