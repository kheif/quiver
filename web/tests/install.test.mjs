// The install commands on the website must be the ones the README documents,
// and point to scripts that exist.
import { test } from 'node:test';
import assert from 'node:assert/strict';
import { existsSync, readFileSync } from 'node:fs';
import { fileURLToPath } from 'node:url';

const root = fileURLToPath(new URL('../../', import.meta.url));
const readme = readFileSync(`${root}README.md`, 'utf8');
const install = JSON.parse(readFileSync(new URL('../src/data/install.json', import.meta.url), 'utf8'));

for (const method of install.methods) {
	test(`${method.label}: the command is in the README`, () => {
		assert.ok(readme.includes(method.command), `README.md does not contain: ${method.command}`);
	});
}

test('the install scripts exist at the repository root', () => {
	for (const script of ['install.sh', 'install.ps1']) {
		assert.ok(existsSync(`${root}${script}`), `${script} is missing`);
	}
});

test('every source link points to the repository or npm', () => {
	for (const method of install.methods) {
		assert.match(method.source, /^https:\/\/(github\.com\/kheif\/quiver|www\.npmjs\.com\/package\/@kheif\/quiver)/);
	}
});
