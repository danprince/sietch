import { ok } from "node:assert";
import { createHash } from "node:crypto";
import { unified } from "unified";
import { visit } from "unist-util-visit";
import { VFile } from "vfile";
import remarkParse from "remark-parse";
import remarkRehype from "remark-rehype";
import remarkGfm from "remark-gfm";
import rehypeStringify from "rehype-stringify";
import rehypePrettyCode from "rehype-pretty-code";
import rehypeSlug from "rehype-slug";
import rehypeAutolinkHeadings from "rehype-autolink-headings";
import rehypeExternalLinks from "rehype-external-links";
import rehypeRaw from "rehype-raw";
import { join, relative } from "node:path";
import type { Island, Site } from "./builder";

interface MarkdownOptions {
  rootDir: string;
}

export function createMarkdownProcessor(options: MarkdownOptions) {
  return unified()
    .use(remarkParse)
    .use(remarkGfm)
    .use(remarkRehype, { allowDangerousHtml: true })
    .use(rehypeSlug)
    .use(rehypeAutolinkHeadings, { behavior: "wrap" })
    .use(rehypeExternalLinks, { target: "_blank", rel: "noopener noreferrer" })
    .use(rehypePrettyCode, {
      theme: "css-variables",
      onVisitHighlightedWord(node) {
        node.tagName = "mark";
      },
      onVisitHighlightedLine(node) {
        node.tagName = "mark";
      },
    })
    .use(rehypeRaw)
    .use(rehypeIslands, options)
    .use(rehypeStringify, { allowDangerousHtml: true });
}

function rehypeIslands(options: MarkdownOptions): any {
  return (tree: any, file: VFile) => {
    let islands: Island[] = file.data.islands = [];
    let counter = 0;

    visit(tree, "element", (node, index, parent) => {
      if (node.tagName !== "island") return;
      let { src, hydrate, ...props } = node.properties;

      ok(typeof index === "number", `Can't parse island at non-numeric index`);

      ok(
        // These components will not hydrate
        hydrate === undefined ||
        // These components will hydrate on load
        hydrate === "" ||
        // These components will not prerender
        hydrate === "clientOnly",
        `Invalid hydration mode: ${hydrate}`,
      );

      // Attempt to parse html attributes as JSON to make it easier to pass
      // literal values in from markdown files.
      for (let key in props) {
        try {
          props[key] = JSON.parse(props[key]);
        } catch {}
      }

      // Use a stable hash for island IDs so that the bundler's content hash
      // doesn't change across rebuilds.
      let id = "_" + createHash("md5")
        .update(`${file.path}-${src}-${counter++}`)
        .digest("hex")
        .slice(0, 7);

      if (src[0] === ".") {
        // If the import is relative then make it relative to the root dir of
        // the site instead, so that we can use a consistent resolveDir later.
        src = "./" + relative(options.rootDir, join(file.dirname!, src));
      }

      let island: Island = {
        id,
        src,
        hydrate: hydrate !== undefined,
        clientOnly: hydrate === "clientOnly",
        props,
      };

      islands.push(island);

      let comment = { type: "comment", value: `island:${island.id}` };

      if (island.hydrate) {
        parent.children[index] = {
          type: "element",
          tagName: "span",
          properties: { "data-island": island.id },
          children: island.clientOnly ? [] : [comment],
        }
      } else {
        parent.children[index] = comment;
      }
    });
  };
}
