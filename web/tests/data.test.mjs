// The data the home page shows: assistants, the demo session and releases.
import { test } from 'node:test';
import assert from 'node:assert/strict';
import { existsSync, readdirSync, readFileSync, statSync } from 'node:fs';
import { join } from 'node:path';
import { fileURLToPath } from 'node:url';

const read = (path) => JSON.parse(readFileSync(new URL(path, import.meta.url), 'utf8'));
const assistants = read('../src/data/assistants.json');
const demo = read('../src/data/demo.json');
const docs = (path) => readFileSync(new URL(`../src/content/docs/docs/${path}`, import.meta.url), 'utf8');

test('every assistant has a known delivery level and a unique id', () => {
	const levels = new Set(['hook', 'instructions', 'on-request', 'planned']);
	const marks = new Set(['coming-soon', 'planned', 'next-release']);
	assert.equal(new Set(assistants.map((a) => a.id)).size, assistants.length);
	for (const a of assistants) {
		assert.ok(levels.has(a.level), `${a.id}: level ${a.level}`);
		for (const m of a.marks ?? []) assert.ok(marks.has(m), `${a.id}: mark ${m}`);
		assert.ok(a.name && a.label, `${a.id}: name and label`);
	}
});

test('the assistants page names every assistant the home page shows', () => {
	const page = docs('assistants.md');
	for (const a of assistants) assert.ok(page.includes(a.name), `assistants.md does not name ${a.name}`);
});

test('the demo has the folders and teams the pickers offer', () => {
	assert.deepEqual(Object.keys(demo.folders), ['checkout-web', 'api', 'none']);
	assert.deepEqual(Object.keys(demo.teams), ['ui', 'backend']);
});

test('every skill that can be always on has a text to show', () => {
	const canBeOn = [
		...demo.company.filter((s) => s.always).map((s) => s.id),
		...demo.you.filter((s) => s.always).map((s) => s.id),
		...Object.values(demo.teams).map((t) => t.skill),
	];
	for (const id of canBeOn) assert.ok(demo.texts[id], `no text for ${id}`);
});

test('site.ts lists the releases newest first', () => {
	const site = readFileSync(new URL('../src/data/site.ts', import.meta.url), 'utf8');
	const tags = [...site.matchAll(/tag: '(v[\d.]+)'/g)].map((m) => m[1]);
	assert.deepEqual(tags, ['v1.4.0', 'v1.3.0', 'v1.2.0', 'v1.1.0', 'v1.0.0']);
	assert.equal((site.match(/summary: '/g) ?? []).length, tags.length, 'a release has no summary');
	assert.ok(existsSync(new URL('../src/data/site.ts', import.meta.url)));
});

test('no docs page but the generated CLI reference names quiver mcp install', () => {
	const root = fileURLToPath(new URL('../src/content/docs/docs/', import.meta.url));
	const files = (dir) =>
		readdirSync(dir).flatMap((n) => (statSync(join(dir, n)).isDirectory() ? files(join(dir, n)) : [join(dir, n)]));
	for (const f of files(root)) {
		if (f.endsWith('reference/cli.md')) continue;
		// A Next release note, as a ::: block or an Aside, may say what an earlier release's users run instead.
		const body = readFileSync(f, 'utf8')
			.replace(/:::note\[Next release\][\s\S]*?:::/g, '')
			.replace(/<Aside [^>]*title="Next release"[^>]*>[\s\S]*?<\/Aside>/g, '');
		assert.ok(!/quiver mcp (un)?install/.test(body), `${f} names quiver mcp install`);
	}
});

test('assistant labels say what works today', () => {
	const byId = Object.fromEntries(assistants.map((a) => [a.id, a]));
	assert.match(byId['claude-desktop'].label, /when you ask/i);
	for (const a of assistants.filter((x) => x.level === 'hook')) assert.equal(a.label, 'Skills and always-on', a.id);
	for (const a of assistants) assert.ok(!(a.marks ?? []).includes('next-release'), `${a.id}: next-release comes from site.ts`);
});

test('the introduction does not mark commands of the latest release', () => {
	const intro = docs('introduction.md');
	for (const cmd of ['quiver pin', 'quiver fork', 'quiver sync', 'quiver update']) {
		const line = intro.split('\n').find((l) => l.includes(cmd));
		assert.ok(line && !/next release/i.test(line), `${cmd} is missing or still marked`);
	}
});

test('every logo belongs to an assistant on the home page', () => {
	const src = readFileSync(new URL('../src/data/logos.ts', import.meta.url), 'utf8');
	const ids = [...src.matchAll(/^\t'([\w-]+)': \{/gm)].map((m) => m[1]);
	assert.ok(ids.length > 0);
	for (const id of ids) assert.ok(assistants.some((a) => a.id === id), `${id} is not an assistant`);
});

test('the install page opens the Windows tab first on Windows', () => {
	const page = docs('installation.mdx');
	const script = readFileSync(new URL('../src/components/WindowsFirst.astro', import.meta.url), 'utf8');
	assert.match(page, /<WindowsFirst \/>[\s\S]*<Tabs syncKey="install">/);
	assert.ok(script.includes("'starlight-synced-tabs__install'"));
	assert.match(page, /<TabItem label="Windows">/);
});
