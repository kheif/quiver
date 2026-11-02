// What a session gets in the home page's sample library: used on the server
// for the default state and in the browser when a picker changes.
import demo from '../data/demo.json';

export type Team = keyof typeof demo.teams;
export type Folder = keyof typeof demo.folders;

export interface Row {
	id: string;
	layer: string;
	always: boolean;
}

/** The skills served in `folder` for a member of `team`: always-on first, then the rest, each in layer order. */
export function served(team: Team, folder: Folder): Row[] {
	const t = demo.teams[team];
	// The company and team layers come with Quiver for teams.
	const planned = ' · planned';
	const rows: Row[] = [
		...demo.company.filter((s) => s.always).map((s) => ({ id: s.id, layer: `company${planned}`, always: true })),
		{ id: t.skill, layer: `${t.name}${planned}`, always: true },
		{ id: t.other, layer: `${t.name}${planned}`, always: false },
		...demo.you.map((s) => ({ id: s.id, layer: 'you', always: s.always })),
		...demo.folders[folder].map((id) => ({ id, layer: folder, always: false })),
	];
	return [...rows.filter((r) => r.always), ...rows.filter((r) => !r.always)];
}

/** The folder's path as the panel shows it. */
export function where(folder: Folder): string {
	return folder === 'none' ? '~/notes' : `~/code/${folder}`;
}

/** The block an assistant has before the first message, for the given always-on skills, as the session-start hook prints it. */
export function alwaysBlock(ids: string[]): string {
	if (ids.length === 0) return '(nothing before the first message)';
	const texts = demo.texts as Record<string, string>;
	const head = 'Always-on skills from Quiver. Follow each one for the whole conversation.';
	return `<quiver-always-on>\n${head}\n\n${ids.map((id) => `## ${id}\n${texts[id]}`).join('\n\n')}\n</quiver-always-on>`;
}
