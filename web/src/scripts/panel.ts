// The recorded panel (data/panel.json), turned into HTML for the website.
import { ansiToHtml } from './ansi.mjs';
import tour from '../data/panel.json';

export interface Event {
	key?: string;
	id?: string;
	caption?: string;
	wait: number;
	/** The screen's lines, as indexes into `lines`. */
	frame: number[];
}

/** Every distinct line of every screen, as HTML. Lines carry no style over to the next. */
export const lines: string[] = tour.lines.map(ansiToHtml);

export const events: Event[] = tour.events;

/** One screen as HTML for a `<pre>`. */
export function screen(event: Event): string {
	return event.frame.map((i) => lines[i]).join('\n');
}

/** The screen named by `id`, such as `projects`. */
export function screenHtml(id: string): string {
	const event = events.find((e) => e.id === id) ?? events[0];
	return screen(event);
}

/** What the story says on the screen named by `id`. */
export function captionOf(id: string): string {
	return (events.find((e) => e.id === id) ?? events[0]).caption ?? '';
}
