// knowledgeIndex — M-05 surface (ii): the `_knowledge/` freeform background folder.
// Recursively lists `_knowledge/**/*.md` (minus the profile file itself), returns per-file
// metadata (path, bytes, words) and file content up to a bounded byte budget; over-budget
// files are returned as index entries only (content omitted). Deterministic (path-sorted).

import { readdirSync, readFileSync, statSync } from "node:fs";
import { join, relative, sep } from "node:path";

/** Default content byte budget: bounds what is auto-loaded so a large folder cannot blow the
 *  context window (M-05: "a size/word budget caps what is auto-loaded"). */
export const DEFAULT_KNOWLEDGE_BUDGET = 65536; // 64 KiB

// Files that are the profile itself, not background prose — excluded from the index.
const PROFILE_EXCLUDE = /^profile(\.example)?\.(ya?ml|json|md)$/i;

export interface KnowledgeEntry {
  /** POSIX-style path relative to the knowledge root. */
  path: string;
  bytes: number;
  words: number;
  contentIncluded: boolean;
  /** Present only when contentIncluded is true. */
  content?: string;
}

export interface KnowledgeIndex {
  root: string;
  budget: number;
  fileCount: number;
  bytesIncluded: number;
  entries: KnowledgeEntry[];
}

/**
 * knowledgeIndex indexes root (a `_knowledge` directory). Files are visited in deterministic
 * path order; content is included while the cumulative included-byte total stays within budget,
 * otherwise the entry carries metadata only (contentIncluded=false). A missing/inaccessible
 * root yields an empty index rather than an error.
 */
export function knowledgeIndex(root: string, budget: number = DEFAULT_KNOWLEDGE_BUDGET): KnowledgeIndex {
  const files = collectMarkdown(root).sort((a, b) => (a < b ? -1 : a > b ? 1 : 0));
  const entries: KnowledgeEntry[] = [];
  let bytesIncluded = 0;
  for (const abs of files) {
    let content: string;
    try {
      content = readFileSync(abs, "utf8");
    } catch {
      continue; // unreadable file — skip rather than fail the whole index
    }
    const bytes = Buffer.byteLength(content, "utf8");
    const words = countWords(content);
    const rel = toPosix(relative(root, abs));
    const fits = bytesIncluded + bytes <= budget;
    if (fits) {
      bytesIncluded += bytes;
      entries.push({ path: rel, bytes, words, contentIncluded: true, content });
    } else {
      entries.push({ path: rel, bytes, words, contentIncluded: false });
    }
  }
  return { root, budget, fileCount: entries.length, bytesIncluded, entries };
}

function collectMarkdown(dir: string): string[] {
  let names: string[];
  try {
    names = readdirSync(dir);
  } catch {
    return [];
  }
  const out: string[] = [];
  for (const name of names) {
    const abs = join(dir, name);
    let isDir = false;
    try {
      isDir = statSync(abs).isDirectory();
    } catch {
      continue;
    }
    if (isDir) {
      out.push(...collectMarkdown(abs));
    } else if (name.toLowerCase().endsWith(".md") && !PROFILE_EXCLUDE.test(name)) {
      out.push(abs);
    }
  }
  return out;
}

function countWords(text: string): number {
  const t = text.trim();
  if (t === "") return 0;
  return t.split(/\s+/).length;
}

function toPosix(p: string): string {
  return sep === "/" ? p : p.split(sep).join("/");
}
