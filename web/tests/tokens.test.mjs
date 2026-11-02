// Every pair of text and ground the site uses stays readable in both themes.
import { test } from 'node:test';
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';

const css = readFileSync(new URL('../src/styles/tokens.css', import.meta.url), 'utf8');

/** The custom properties declared in the first block that opens with `selector {`. */
function block(selector) {
	const start = css.indexOf(`${selector} {`);
	assert.ok(start >= 0, `tokens.css has no block for ${selector}`);
	const body = css.slice(start, css.indexOf('}', start));
	return Object.fromEntries([...body.matchAll(/(--q-[\w-]+):\s*([^;]+);/g)].map((m) => [m[1], m[2].trim()]));
}

const light = block(':root');
const dark = { ...light, ...block(":root[data-theme='dark']") };

function luminance(hex) {
	const m = hex.match(/^#([0-9a-f]{6})$/i);
	assert.ok(m, `not a #rrggbb colour: ${hex}`);
	const [r, g, b] = [0, 2, 4].map((i) => {
		const c = parseInt(m[1].slice(i, i + 2), 16) / 255;
		return c <= 0.03928 ? c / 12.92 : ((c + 0.055) / 1.055) ** 2.4;
	});
	return 0.2126 * r + 0.7152 * g + 0.0722 * b;
}

function contrast(a, b) {
	const [hi, lo] = [luminance(a), luminance(b)].sort((x, y) => y - x);
	return (hi + 0.05) / (lo + 0.05);
}

const text = [
	['--q-text', '--q-bg'],
	['--q-text', '--q-surface'],
	['--q-text-2', '--q-bg'],
	['--q-text-2', '--q-surface'],
	['--q-accent-ink', '--q-accent'],
	['--q-panel-ink', '--q-panel'],
	['--q-accent', '--q-bg'],
];

for (const [name, theme] of [['light', light], ['dark', dark]]) {
	test(`${name} theme: text keeps 4.5:1 on its ground`, () => {
		for (const [fg, bg] of text) {
			const ratio = contrast(theme[fg], theme[bg]);
			assert.ok(ratio >= 4.5, `${fg} on ${bg}: ${ratio.toFixed(2)}:1`);
		}
	});
	test(`${name} theme: the accent keeps 3:1 as a fill`, () => {
		assert.ok(contrast(theme['--q-accent'], theme['--q-bg']) >= 3);
		assert.ok(contrast(theme['--q-accent'], theme['--q-surface']) >= 3);
	});
}

test('docs tables keep the first column whole, so names do not break mid-word', () => {
	const sl = readFileSync(new URL('../src/styles/starlight.css', import.meta.url), 'utf8');
	assert.match(sl, /:is\(td, th\):first-child\s*\{[^}]*overflow-wrap:\s*normal/);
});

test('buttons and links on coloured grounds keep a visible focus ring', () => {
	const base = readFileSync(new URL('../src/styles/base.css', import.meta.url), 'utf8');
	assert.match(base, /\.btn:focus-visible\s*\{[^}]*outline-offset:\s*-/, 'the stair cut clips an outer ring');
	assert.match(base, /accent-ground[^{]*:focus-visible\s*\{[^}]*--q-accent-ink/, 'no ring colour for the accent ground');
	assert.match(base, /panel-ground[^{]*:focus-visible\s*\{[^}]*--q-panel-ink/, 'no ring colour for the dark panel');
});

test('the white panel on the accent field keeps the accent ring, and an off switch stays visible', () => {
	const base = readFileSync(new URL('../src/styles/base.css', import.meta.url), 'utf8');
	const accent = base.search(/\.accent-ground :focus-visible/);
	const plain = base.search(/\.plain-ground :focus-visible\s*\{[^}]*--q-accent\)/);
	assert.ok(plain > accent, 'no later ring rule for a plain panel inside the accent field');
	const tryCss = readFileSync(new URL('../src/components/TrySession.astro', import.meta.url), 'utf8');
	assert.match(tryCss, /\.sw\s*\{[^}]*background:\s*var\(--q-text-2\)/, 'the off track must keep 3:1 on its ground');
});
