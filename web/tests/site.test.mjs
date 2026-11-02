// Checks the built site in dist/: every internal link and asset resolves under
// the base path, and every page has the metadata search and social cards need.
// Run `npm run build` first.
import { test } from 'node:test';
import assert from 'node:assert/strict';
import { existsSync, readdirSync, readFileSync, statSync } from 'node:fs';
import { join, relative } from 'node:path';
import { fileURLToPath } from 'node:url';
import rehypeBaseLinks from '../src/plugins/rehype-base-links.mjs';

const dist = fileURLToPath(new URL('../dist/', import.meta.url));
const base = (process.env.BASE_PATH ?? '/quiver').replace(/\/$/, '');
const built = existsSync(dist);

function pages(dir = dist) {
	return readdirSync(dir).flatMap((name) => {
		const path = join(dir, name);
		if (statSync(path).isDirectory()) return name === 'pagefind' ? [] : pages(path);
		return name.endsWith('.html') ? [path] : [];
	});
}

/** The file that a site path such as /quiver/docs/ is served from. */
function resolve(path) {
	const clean = decodeURI(path.split(/[?#]/)[0]).slice(base.length);
	const file = join(dist, clean);
	if (clean.endsWith('/')) return join(file, 'index.html');
	return file;
}

test('rehype-base-links prefixes root-relative links only', () => {
	const tree = {
		type: 'root',
		children: ['/docs/a/', '/quiver/docs/b/', 'https://x.dev/', '#top', '//cdn.dev/x', '/'].map((href) => ({
			type: 'element',
			tagName: 'a',
			properties: { href },
			children: [],
		})),
	};
	rehypeBaseLinks({ base: '/quiver' })(tree);
	assert.deepEqual(
		tree.children.map((n) => n.properties.href),
		['/quiver/docs/a/', '/quiver/docs/b/', 'https://x.dev/', '#top', '//cdn.dev/x', '/quiver/'],
	);
});

test('internal links and assets resolve under the base path', { skip: !built && 'run npm run build first' }, () => {
	const broken = [];
	for (const page of pages()) {
		const html = readFileSync(page, 'utf8');
		for (const [, attr, value] of html.matchAll(/\s(href|src)="([^"]+)"/g)) {
			if (!value.startsWith('/') || value.startsWith('//')) continue;
			const where = relative(dist, page);
			if (!value.startsWith(`${base}/`)) {
				broken.push(`${where}: ${attr}="${value}" lacks the base path ${base}/`);
			} else if (!existsSync(resolve(value))) {
				broken.push(`${where}: ${attr}="${value}" does not exist`);
			}
		}
	}
	assert.deepEqual(broken, []);
});

test('anchors point to an element on the target page', { skip: !built && 'run npm run build first' }, () => {
	const ids = (file) => new Set([...readFileSync(file, 'utf8').matchAll(/\sid="([^"]+)"/g)].map((m) => m[1]));
	const broken = [];
	for (const page of pages()) {
		const html = readFileSync(page, 'utf8');
		for (const [, path, id] of html.matchAll(/\shref="([^"#]*)#([^"]+)"/g)) {
			if (path && !path.startsWith(`${base}/`)) continue;
			const target = path ? resolve(path) : page;
			if (!existsSync(target) || !ids(target).has(decodeURIComponent(id))) {
				broken.push(`${relative(dist, page)}: ${path}#${id}`);
			}
		}
	}
	assert.deepEqual(broken, []);
});

test('every page has a title, a description and a canonical URL', { skip: !built && 'run npm run build first' }, () => {
	const missing = [];
	for (const page of pages()) {
		const html = readFileSync(page, 'utf8');
		const where = relative(dist, page);
		if (!/<title>[^<]+<\/title>/.test(html)) missing.push(`${where}: title`);
		if (!/<meta name="description" content="[^"]+"/.test(html)) missing.push(`${where}: description`);
		if (where !== '404.html' && !/<link rel="canonical" href="https?:\/\/[^"]+"/.test(html)) {
			missing.push(`${where}: canonical`);
		}
		if (!/<meta property="og:image" content="https?:\/\/[^"]+\/og\.png"/.test(html)) missing.push(`${where}: og:image`);
	}
	assert.deepEqual(missing, []);
	assert.ok(existsSync(join(dist, 'og.png')), 'og.png is missing');
});
