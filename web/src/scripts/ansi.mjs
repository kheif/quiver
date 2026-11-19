// Turns the panel's styled text (SGR escape sequences) into HTML for the
// website's terminal. Only what the panel uses is read: bold, faint,
// underline, reverse, and its one accent colour.

const ACCENT = '107;140;255';

/** @param {string} s */
function escape(s) {
	return s.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;');
}

/**
 * @param {string} text styled text with `ESC [ … m` sequences
 * @returns {string} HTML for a `<pre>`: styled runs are `<span class="…">`
 */
export function ansiToHtml(text) {
	const state = { b: false, f: false, u: false, r: false, a: false };
	let out = '';
	let last = 0;
	const sgr = /\x1b\[([0-9;:]*)m/g;
	/** @param {string} chunk */
	const emit = (chunk) => {
		if (chunk === '') return;
		const cls = Object.entries(state)
			.filter(([, on]) => on)
			.map(([k]) => k)
			.join(' ');
		out += cls ? `<span class="${cls}">${escape(chunk)}</span>` : escape(chunk);
	};
	for (const m of text.matchAll(sgr)) {
		emit(text.slice(last, m.index));
		last = m.index + m[0].length;
		const codes = m[1] === '' ? ['0'] : m[1].split(';');
		for (let i = 0; i < codes.length; i++) {
			const c = codes[i];
			if (c === '0') Object.assign(state, { b: false, f: false, u: false, r: false, a: false });
			else if (c === '1') state.b = true;
			else if (c === '2') state.f = true;
			else if (c === '4') state.u = true;
			else if (c === '7') state.r = true;
			else if (c === '22') Object.assign(state, { b: false, f: false });
			else if (c === '24') state.u = false;
			else if (c === '27') state.r = false;
			else if (c === '38' && codes[i + 1] === '2') {
				state.a = codes.slice(i + 2, i + 5).join(';') === ACCENT;
				i += 4;
			} else if (c === '39' || (c >= '30' && c <= '37')) state.a = false;
		}
	}
	emit(text.slice(last));
	return out;
}
