// Adds the site's base path to root-relative links in Markdown, so docs can
// link to /docs/quickstart/ and still work under /quiver/ on GitHub Pages or
// at the root of a custom domain.
export default function rehypeBaseLinks({ base = '/' } = {}) {
	const prefix = base.replace(/\/$/, '');
	const visit = (node) => {
		if (node.type === 'element' && node.tagName === 'a') {
			const href = node.properties?.href;
			if (
				prefix &&
				typeof href === 'string' &&
				href.startsWith('/') &&
				!href.startsWith('//') &&
				href !== prefix &&
				!href.startsWith(`${prefix}/`)
			) {
				node.properties.href = prefix + href;
			}
		}
		for (const child of node.children ?? []) visit(child);
	};
	return (tree) => visit(tree);
}
