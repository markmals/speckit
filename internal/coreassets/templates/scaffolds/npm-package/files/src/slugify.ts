// Convert text into a URL-safe slug: lowercase, collapse every run of
// non-alphanumeric characters to a single hyphen, and trim leading/trailing hyphens.
export function slugify(text: string): string {
    return text
        .toLowerCase()
        .replace(/[^a-z0-9]+/g, "-")
        .replace(/^-+|-+$/g, "");
}
