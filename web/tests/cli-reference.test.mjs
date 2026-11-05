import { test } from 'node:test';
import assert from 'node:assert/strict';
import { descriptionToMarkdown, parseHelp } from '../scripts/cli-reference.mjs';

const help = `Assign skills to this project, or to every folder with --global.

  quiver import             the .claude/skills of this project

Usage:
  quiver add <id>... [flags]

Examples:
  # Assign two skills
  quiver add pdf anthropic/docx

Available Commands:
  install     Set up the MCP server and the session-start hooks in your assistants

Flags:
      --always           also make the skills always-on
  -h, --help             help for add
      --project string   project folder (default: the current folder)
  -y, --yes              register a new project without asking

Global Flags:
  -v, --verbose   print more details

Use "quiver add [command] --help" for more information about a command.
`;

test('parseHelp splits cobra help into sections', () => {
	const h = parseHelp(help);
	assert.deepEqual(h.usage, ['quiver add <id>... [flags]']);
	assert.deepEqual(h.examples, ['# Assign two skills', 'quiver add pdf anthropic/docx']);
	assert.deepEqual(h.commands, [{ name: 'install', summary: 'Set up the MCP server and the session-start hooks in your assistants' }]);
	assert.deepEqual(
		h.flags.map((f) => [f.short, f.long, f.type]),
		[
			[undefined, '--always', undefined],
			[undefined, '--project', 'string'],
			['-y', '--yes', undefined],
		],
	);
});

test('descriptionToMarkdown keeps indented lines as code and escapes prose', () => {
	const md = descriptionToMarkdown(parseHelp(help).description);
	assert.equal(
		md,
		'Assign skills to this project, or to every folder with --global.\n\n```text\nquiver import             the .claude/skills of this project\n```',
	);
	assert.equal(descriptionToMarkdown(['Use <id> and *this*.']), 'Use \\<id\\> and \\*this\\*.');
});
