export const repo = 'https://github.com/kheif/quiver';

export const links = {
	repo,
	releases: `${repo}/releases`,
	issues: `${repo}/issues`,
	newIssue: `${repo}/issues/new/choose`,
	license: `${repo}/blob/main/LICENSE`,
	contributing: `${repo}/blob/main/CONTRIBUTING.md`,
	security: `${repo}/blob/main/SECURITY.md`,
	website: `${repo}/tree/main/web`,
};

/** Joins a site path to the base path, such as /quiver/. */
export function url(path = '/'): string {
	const base = import.meta.env.BASE_URL.replace(/\/$/, '');
	return `${base}/${path.replace(/^\//, '')}`;
}

/**
 * Assistants supported on the main branch but not yet in the latest release.
 * The home page's assistant tiles mark them; empty the set after the next release.
 */
export const latestRelease = 'v1.4.0';
export const unreleased = new Set<string>([]);

/** Every release, newest first. The home page shows the latest four. */
export const releases: { tag: string; date: string; summary?: string }[] = [
	{
		tag: 'v1.4.0',
		date: '4 November 2026',
		summary: 'session-start hooks that deliver always-on skills in full, quiver install, VS Code Copilot',
	},
	{
		tag: 'v1.3.0',
		date: '2 November 2026',
		summary: 'sync between machines, update preview, pin and fork, Codex and Antigravity',
	},
	{
		tag: 'v1.2.0',
		date: '31 October 2026',
		summary: 'show and search, a guided first run, Cursor and Claude Desktop, a Windows installer',
	},
	{
		tag: 'v1.1.0',
		date: '29 October 2026',
		summary: 'import one skill or a whole folder, skills written as a single markdown file',
	},
	{
		tag: 'v1.0.0',
		date: '28 October 2026',
		summary: 'the first release: library, layers, sources and the MCP server for Claude Code',
	},
];
