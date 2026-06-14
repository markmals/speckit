import { EditorContent, useEditor } from "@tiptap/react";
import StarterKit from "@tiptap/starter-kit";

import { cx } from "@/lib/cva";

export interface RichTextEditorProps {
    /** Initial document content, as HTML. */
    content?: string;
    /** Whether the document can be edited. */
    editable?: boolean;
    /** Extra classes merged onto the editor surface. */
    className?: string;
    /** Called with the document's HTML on every change. */
    onUpdate?: (html: string) => void;
}

/**
 * A minimal rich-text editor built on Tiptap + StarterKit, styled with rac-ui's
 * cva/Tailwind tokens. Drop it into a route or wrap it in a form field; read
 * changes through `onUpdate`.
 */
export function RichTextEditor({
    className,
    content = "",
    editable = true,
    onUpdate,
}: RichTextEditorProps) {
    let editor = useEditor({
        extensions: [StarterKit],
        content,
        editable,
        // TanStack Start renders on the server first; defer the editor's initial
        // paint to the client so the SSR markup and the hydrated tree match
        // (Tiptap requires this in SSR frameworks).
        immediatelyRender: false,
        editorProps: {
            attributes: {
                class: cx(
                    "min-h-40 w-full rounded-md border border-neutral-300 bg-white px-3 py-2 text-sm",
                    "focus:outline-none focus-visible:ring-2 focus-visible:ring-neutral-900",
                    className,
                ),
            },
        },
        onUpdate: ({ editor: instance }) => onUpdate?.(instance.getHTML()),
    });

    return <EditorContent editor={editor} />;
}
