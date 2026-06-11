// SPEC: story.todo.toggle — web reference implementation
export interface Todo {
	label: string;
	done: boolean;
}

export function toggle(todo: Todo): Todo {
	if (todo.label.trim() === "") throw new Error("empty label");
	return { ...todo, done: !todo.done };
}
