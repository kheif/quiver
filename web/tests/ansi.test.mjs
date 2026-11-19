import { test } from 'node:test';
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import { ansiToHtml } from '../src/scripts/ansi.mjs';

test('plain text is escaped and kept', () => {
	assert.equal(ansiToHtml('a <b> & c'), 'a &lt;b&gt; &amp; c');
});

test('bold, faint and the accent colour become classes, and a reset ends them', () => {
	const html = ansiToHtml('\x1b[1;38;2;107;140;255mhi\x1b[m \x1b[2mdim\x1b[m end');
	assert.equal(html, '<span class="b a">hi</span> <span class="f">dim</span> end');
});

test('a colour other than the accent is not the accent', () => {
	assert.equal(ansiToHtml('\x1b[38;2;1;2;3mx\x1b[m'), 'x');
});

test('every recorded screen converts, keeps its text, and no line leaves a style on', () => {
	const tour = JSON.parse(readFileSync(new URL('../src/data/panel.json', import.meta.url), 'utf8'));
	assert.ok(tour.events.length >= 10);
	for (const line of tour.lines) {
		const html = ansiToHtml(line);
		assert.ok(!html.includes('\x1b'), 'an unread escape');
		const plain = html.replace(/<[^>]+>/g, '').replace(/&lt;/g, '<').replace(/&gt;/g, '>').replace(/&amp;/g, '&');
		assert.equal(plain, line.replace(/\x1b\[[0-9;:]*m/g, ''));
		const last = [...line.matchAll(/\x1b\[([0-9;:]*)m/g)].at(-1);
		assert.ok(!last || last[1] === '' || last[1] === '0', `a style runs past the end of ${JSON.stringify(line)}`);
	}
	for (const e of tour.events) {
		assert.ok(e.frame.length === tour.rows && e.frame.every((i) => tour.lines[i] !== undefined));
	}
	const ids = tour.events.map((e) => e.id).filter(Boolean);
	for (const id of ['skills', 'search', 'assign', 'projects', 'sources', 'update', 'health']) assert.ok(ids.includes(id), id);
});
