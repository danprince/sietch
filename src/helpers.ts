import { readdir, readFile, stat } from "node:fs/promises";
import { join } from "node:path";
import { bold } from "picocolors";

/**
 * @internal
 */
export function createPageId(str: string): string {
  return str
    .toLowerCase()
    .replace(/\.md$/, "")
    .replace(/[^a-z0-9]+/g, "_")
    .replace(/_+/, "_");
}

/**
 * @internal
 */
export function slugify(str: string): string {
  return str
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, "_")
    .replace(/_+/, "_");
}

/**
 * @internal
 */
export function isIgnored(file: string): boolean {
  return (
    file[0] === "_" ||
    file[0] === "." ||
    file === "node_modules"
  );
}

/**
 * @internal
 */
export async function scan(dir: string, ext: string): Promise<string[]> {
  let entries = await readdir(dir, { withFileTypes: true });

  let files = await Promise.all(
    entries
      .filter(entry => !isIgnored(entry.name))
      .filter(entry => entry.isDirectory() || entry.name.endsWith(ext))
      .map(entry => {
        let name = join(dir, entry.name);
        return entry.isDirectory() ? scan(name, ext) : name;
      }),
  );

  return files.flat();
}

/**
 * @internal
 */
export async function readJsonSafe<T = any>(path: string): Promise<T | undefined> {
  try {
    await stat(path);
  } catch (err) {
    return undefined;
  }

  let contents = await readFile(path, "utf8");
  return JSON.parse(contents);
}

/**
 * @internal
 */
export function debounce<Args extends any[]>(
  func: (...args: Args) => any,
  delay: number,
) {
  let timeout: any;
  return (...args: Args) => {
    clearTimeout(timeout);
    timeout = setTimeout(() => func(...args), delay);
  };
}

export async function time<T>(label: string, promise: Promise<T>): Promise<T> {
  let start = performance.now();
  let result = await promise;
  let end = performance.now();
  let ms = Math.floor((end - start) * 100) / 100;
  console.log(`${label} ${bold(ms + "ms")}`);
  return result;
}
