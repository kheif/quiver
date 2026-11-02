// Makes tables focusable, so keyboard users can scroll a table that is wider
// than a narrow screen.
export default function rehypeFocusableTables() {
	const visit = (node) => {
		if (node.type === 'element' && node.tagName === 'table') {
			node.properties = { ...node.properties, tabIndex: 0 };
		}
		for (const child of node.children ?? []) visit(child);
	};
	return (tree) => visit(tree);
}
