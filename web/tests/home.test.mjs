// The built home page: its story, and that every live panel reads without
// scripts. Run `npm run build` first.
import { test } from 'node:test';
import assert from 'node:assert/strict';
import { existsSync, readFileSync } from 'node:fs';

const file = new URL('../dist/index.html', import.meta.url);
const built = existsSync(file);
const skip = !built && 'run npm run build first';
const html = built ? readFileSync(file, 'utf8') : '';
const text = (s) => s.replace(/<[^>]+>/g, ' ').replace(/&lt;/g, '<').replace(/&gt;/g, '>').replace(/&amp;/g, '&').replace(/\s+/g, ' ');

/** The HTML of the element that carries `marker` (such as id="why"), up to its section's end. */
function part(marker, end = '</section>') {
	const i = html.indexOf(marker);
	assert.ok(i >= 0, `no ${marker} on the page`);
	return html.slice(i, html.indexOf(end, i));
}

test('hero: the headline and an honest lede', { skip }, () => {
	assert.match(html, /<h1[^>]*>Agent skills, managed in layers<\/h1>/);
	assert.ok(text(part('class="hero')).includes('you and the project today, your team and your company next'));
});

test('hero: the layer picker shows its default state without scripts', { skip }, () => {
	const served = part('id="served"', '</ul>');
	assert.equal((served.match(/<li/g) ?? []).length, 6);
	const buttons = part('class="hero').match(/<button[^>]*data-(team|folder)=[^>]*>/g) ?? [];
	assert.equal(buttons.length, 5);
	for (const b of buttons) assert.match(b, /aria-pressed="(true|false)"/);
	assert.match(text(part('class="hero')), /loaded at session start\./);
	assert.doesNotMatch(text(part('class="hero')), /coming soon/i);
});

test('hero: the install command is there to copy', { skip }, () => {
	assert.ok(part('class="hero').includes('npm install -g @kheif/quiver'));
});

test('why: the heading names the problem, and where each assistant keeps its own copy', { skip }, () => {
	assert.match(html, /<h2[^>]*>Your skills are scattered across assistants<\/h2>/);
	const why = part('id="why"');
	for (const place of ['.claude/skills', '.cursor/rules', '.codex/skills', 'AGENTS.md']) assert.ok(why.includes(place), `no ${place}`);
});

test('how it works: three steps and three states, one shown without scripts', { skip }, () => {
	const how = part('id="how"');
	assert.equal((how.match(/class="step[ "]/g) ?? []).length, 3);
	assert.equal((how.match(/data-s="/g) ?? []).length, 3);
	assert.equal((how.match(/class="state on[ "]/g) ?? []).length, 1);
});

test('try one session: the always-on block and the switches read without scripts', { skip }, () => {
	const t = part('id="try"');
	const ctx = t.slice(t.indexOf('id="ctx"'), t.indexOf('</pre>', t.indexOf('id="ctx"')));
	assert.ok(ctx.includes('&lt;quiver-always-on&gt;'), 'no always-on block');
	assert.ok(ctx.includes('## security-review'), 'no security-review');
	const switches = t.match(/<button[^>]*role="switch"[^>]*>/g) ?? [];
	assert.equal(switches.length, 3);
	assert.equal(switches.filter((b) => b.includes('aria-checked="true"')).length, 2);
	assert.match(text(t), /a session-start hook puts this block in the conversation/);
	assert.doesNotMatch(text(t), /coming soon/i);
	assert.ok(text(t).includes('quiver add reply-style --global --always'));
});

test('the page tells its story in order', { skip }, () => {
	const order = ['id="why"', 'id="how"', 'id="try"', 'id="teams"', 'id="assistants"', 'id="open"', 'id="install"'];
	const at = order.map((m) => html.indexOf(m));
	for (const [i, pos] of at.entries()) assert.ok(pos > 0, `no ${order[i]}`);
	assert.deepEqual([...at].sort((a, b) => a - b), at);
});

test('teams: roadmap wording, and no email address in the HTML', { skip }, () => {
	assert.ok(text(part('id="teams"')).includes('On the roadmap'));
	assert.doesNotMatch(html, /[\w.+-]+@[\w-]+\.[a-z]{2,}/i);
});

test('assistants and releases come from the data files', { skip }, () => {
	const tiles = text(part('id="assistants"'));
	const assistants = JSON.parse(readFileSync(new URL('../src/data/assistants.json', import.meta.url), 'utf8'));
	for (const a of assistants) assert.ok(tiles.includes(a.name), `no ${a.name}`);
	const open = text(part('id="open"'));
	for (const tag of ['v1.4.0', 'v1.3.0', 'v1.2.0', 'v1.1.0']) assert.ok(open.includes(tag), `no ${tag}`);
	assert.ok(!open.includes('v1.0.0'), 'more than the latest four releases');
	assert.equal((part('id="open"').match(/<em[ >]/g) ?? []).length, 4, 'a release has no summary');
});

test('no command on the page is the old mcp install', { skip }, () => {
	assert.ok(!text(html).includes('quiver mcp install'));
});

const teamsFile = new URL('../dist/teams/index.html', import.meta.url);
test('the teams page: planned, plain logic, no commands, a way to write', { skip }, () => {
	assert.ok(existsSync(teamsFile), 'no dist/teams/index.html');
	const page = readFileSync(teamsFile, 'utf8');
	assert.match(page, /<h1[^>]*>Quiver for teams<\/h1>/);
	assert.match(text(page), /Planned/);
	assert.ok(!/<pre/.test(page), 'the teams page shows no command blocks');
	assert.ok(!/\$ quiver|>quiver (install|add|source)/.test(page), 'the teams page shows no quiver command');
	assert.ok(page.includes('email-reveal'), 'no way to write');
	for (const layer of ['Company', 'Team', 'You', 'Project']) assert.ok(text(page).includes(layer), `no ${layer} layer`);
});

test('words stay apart where a line ends before inline code', { skip }, () => {
	assert.match(html, /Codex in\s+<code[^>]*>\.codex\/skills/, 'Codex in.codex/skills');
});

test('hero: planned layers are marked on their own rows, and the summary comes before the rows', { skip }, () => {
	const served = part('id="served"', '</ul>');
	assert.match(text(served), /company · planned/);
	assert.match(text(served), /UI team · planned/);
	assert.ok(html.indexOf('id="served-sum"') < html.indexOf('id="served"'), 'summary after the rows');
});

test('next-release tags follow unreleased in site.ts', { skip }, () => {
	const site = readFileSync(new URL('../src/data/site.ts', import.meta.url), 'utf8');
	const ids = JSON.parse(site.match(/unreleased = new Set(?:<\w+>)?\((\[[^\]]*\])\)/)[1].replace(/'/g, '"'));
	for (const id of ids) assert.match(html, new RegExp(`data-assistant="${id}"[^>]*>[\\s\\S]*?tag-next`), `${id} has no Next release tag`);
	assert.equal((html.match(/class="tag-next/g) ?? []).length, ids.length);
});

test('a failed bundle cannot leave the page hidden', { skip }, () => {
	assert.match(html, /quiverRise/, 'no fallback that shows .rise blocks');
});

const teamsHtml = existsSync(teamsFile) ? readFileSync(teamsFile, 'utf8') : '';

test('headings on the home and teams pages do not end in a full stop', { skip }, () => {
	for (const [name, page] of [['home', html], ['teams', teamsHtml]]) {
		for (const h of page.match(/<h[1-3][^>]*>[\s\S]*?<\/h[1-3]>/g) ?? []) {
			assert.ok(!/\.\s*$/.test(text(h).trim()), `${name}: "${text(h).trim()}"`);
		}
	}
});

test('how it works: step titles are headings, without labels like First and Last, and no dead live region', { skip }, () => {
	const how = part('id="how"');
	assert.equal((how.match(/<h3[^>]*class="st[ "]/g) ?? []).length, 3);
	assert.doesNotMatch(text(how), /\b(First|Then|Last),? /);
	assert.ok(!how.includes('aria-live'), 'aria-live on panels that never announce');
	assert.ok(!how.includes('■'), 'a black square is read out by name');
});

test('how it works: the install output is the real format', { skip }, () => {
	const how = text(part('id="how"'));
	assert.match(how, /Registered with Claude Code: quiver mcp/);
});

test('try one session: switches are named by the skill they show, and changes are announced briefly', { skip }, () => {
	const t = part('id="try"');
	for (const b of t.match(/<button[^>]*role="switch"[^>]*>/g) ?? []) {
		assert.ok(!b.includes('aria-label='), `switch with a name that differs from its label: ${b}`);
		assert.match(b, /aria-labelledby="[^"]+"/);
	}
	assert.match(t, /aria-live="polite"/);
	assert.doesNotMatch(text(t), /left side/i);
	assert.match(t, /class="spanel plain-ground/, 'focus rings inside the white panel take the accent ground colour');
});

test('try one session: the always-on block has the header line of the real hook', { skip }, () => {
	assert.ok(text(part('id="try"')).includes('Always-on skills from Quiver. Follow each one for the whole conversation.'));
});

test('hero: always-on marks are text for screen readers, and only a short status is live', { skip }, () => {
	const hero = part('class="hero');
	assert.ok(!/<span[^>]*aria-label=/.test(hero), 'aria-label on a span is not read');
	assert.ok(!part('id="served"', '>').includes('aria-live'), 'the whole list is re-read on each pick');
	assert.match(hero, /id="picker-status"[^>]*aria-live="polite"|aria-live="polite"[^>]*id="picker-status"/);
	assert.match(hero, /class="ui[ "][^>]*role="group"/);
});

test('install: the heading is named without the copy button, and the next step is quiver init', { skip }, () => {
	const install = part('id="install"');
	assert.match(install, /<h2[^>]*aria-labelledby="install-words"/);
	assert.match(text(install), /Then run quiver init/);
});

test('teams: hosting is coming soon, and nothing points to a plan that is not written', { skip }, () => {
	for (const [name, page] of [['home', text(part('id="teams"'))], ['teams page', text(teamsHtml)]]) {
		assert.doesNotMatch(page, /plan is public|let us host|Read the plan/i, name);
		assert.match(page, /hosting[^.]*coming soon|coming soon[^.]*hosting/i, `${name}: hosting is not marked coming soon`);
	}
	assert.match(text(teamsHtml), /What a team can do today/);
	assert.match(text(teamsHtml), /guidance/i, 'the teams page hides that a skill is guidance, not enforcement');
});

test('the email button is hidden without scripts, and the address is never spelled out', { skip }, () => {
	assert.doesNotMatch(html, /gmail dot com/);
	const base = readFileSync(new URL('../src/styles/base.css', import.meta.url), 'utf8');
	assert.match(base, /html:not\(\.js\) \.email-reveal\s*\{[^}]*display:\s*none/);
});

test('the header names Releases as releases', { skip }, () => {
	assert.doesNotMatch(text(part('class="site-header', '</header>')), /Changelog/);
});

test('how it works: panels stay visible without scripts', () => {
	const how = readFileSync(new URL('../src/components/HowSteps.astro', import.meta.url), 'utf8');
	const css = how.slice(how.indexOf('<style>'));
	for (const rule of css.match(/[^{}]+\{[^{}]*opacity:\s*0;[^{}]*\}/g) ?? []) {
		assert.match(rule.split('{')[0], /\.js/, `hidden without scripts: ${rule.split('{')[0].trim()}`);
	}
});
