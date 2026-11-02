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
export const latestRelease = 'v1.2.0';
export const unreleased = new Set(['codex', 'antigravity']);

/** The latest releases, newest first, for the home page. */
export const releases: { tag: string; date: string; summary?: string }[] = [
	{
		tag: 'v1.2.0',
		date: '31 October 2026',
		summary: 'show and search, a guided first run, Cursor and Claude Desktop, a Windows installer',
	},
	{ tag: 'v1.1.0', date: '29 October 2026' },
	{ tag: 'v1.0.0', date: '28 October 2026' },
];
