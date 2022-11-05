import { mkdir, writeFile } from "node:fs/promises";
import { dirname, join, relative } from "node:path";
import { read } from "to-vfile";
import { VFile } from "vfile";
import { Processor } from "unified";
import { matter } from "vfile-matter";
import { build as esbuild, Message, OnLoadResult } from "esbuild";
import { virtualModulesPlugin } from "./plugins";
import { readJsonSafe, scan, createPageId } from "./helpers";
import { renderLayout } from "./layout";
import { createMarkdownProcessor } from "./markdown";

export interface Site {
  url: string;
  rootDir: string;
  cacheDir: string;
  outDir: string;
  pagesDir: string;
  publicDir: string;
  packageJson: Record<string, any> | undefined;
  pages: Page[];
  markdown: Processor;
  env: "development" | "production";
}

export interface Page {
  id: string;
  url: string;
  dir: string;
  inputPath: string;
  outputPath: string;
  contents: string;
  frontmatter: Record<string, any>;
  islands: Island[];
  scripts: string[];
  styles: string[];
}

export interface Island {
  id: string;
  src: string;
  hydrate: boolean;
  props: Record<string, any>;
  clientOnly: boolean;
}

export class EsbuildError extends Error {
  constructor(public messages: Message[]) {
    super("esbuild errors");
  }
}

/**
 * @internal
 */
export async function siteFromDir(rootDir: string): Promise<Site> {
  let pkgJsonFile = join(rootDir, "package.json");
  let pkg = await readJsonSafe(pkgJsonFile) || {};

  return {
    url: pkg.site?.url,
    packageJson: pkg,
    rootDir,
    pagesDir: join(rootDir, pkg.site?.pagesDir || ""),
    cacheDir: join(rootDir, ".cache"),
    outDir: join(rootDir, "_site"),
    publicDir: join(rootDir, "public"),
    pages: [],
    markdown: createMarkdownProcessor({ rootDir }),
    env: process.env.NODE_ENV === "production" ? "production" : "development",
  };
}

async function readPages(site: Site) {
  let files = await scan(site.pagesDir, ".md");
  site.pages = await Promise.all(files.map(file => readPage(site, file)));
}

async function readPage(site: Site, file: string): Promise<Page> {
  let inputPath = relative(site.pagesDir, file);

  let outputPath = inputPath.endsWith("index.md")
    ? inputPath.replace(/\.md$/, ".html")
    : inputPath.replace(/\.md$/, "/index.html");

  let url = outputPath.replace(/index\.html$/, "") || "/";

  let vfile = await read(file);
  matter(vfile, { strip: true });

  return {
    id: createPageId(inputPath),
    inputPath,
    outputPath,
    url,
    dir: dirname(inputPath),
    contents: vfile.toString(),
    frontmatter: vfile.data.matter as any,
    islands: [],
    scripts: [],
    styles: [],
  };
}

async function bundlePages(site: Site) {
  let [_, bundles] = await Promise.all([
    compileClientBundles(site),
    compileStaticBundles(site),
  ]);

  delete require.cache[bundles.islandsFile];
  delete require.cache[bundles.layoutFile];

  let layout = require(bundles.layoutFile).default;
  let islandsHtml = require(bundles.islandsFile).html;

  for (let page of site.pages) {
    for (let island of page.islands) {
      if (!island.clientOnly) {
        let marker = `<!--island:${island.id}-->`;
        let html = islandsHtml[island.id];
        page.contents = page.contents.replace(marker, html);
      }
    }

    page.contents = renderLayout(site, page, layout);
  }
}

async function writePages(site: Site) {
  await Promise.all(site.pages.map(async page => {
    let outFile = join(site.outDir, page.outputPath);
    let outDir = dirname(outFile);
    await mkdir(outDir, { recursive: true });
    await writeFile(outFile, page.contents);
  }));
}

async function renderPages(site: Site) {
  await Promise.all(site.pages.map(page => renderPage(site, page)));
}

async function renderPage(site: Site, page: Page) {
  let vfile = new VFile({
    path: join(site.pagesDir, page.inputPath),
    value: page.contents,
  });

  await site.markdown.process(vfile);
  page.contents = vfile.toString();
  page.islands = vfile.data.islands as Island[];
}

async function compileStaticBundles(site: Site) {
  let islands = site.pages
    .flatMap(page => page.islands)
    .filter(island => !island.clientOnly);

  let result = await esbuild({
    entryPoints: {
      "islands": "static:islands",
      "layout": join(site.rootDir, "layout"),
    },
    target: "node16",
    platform: "node",
    format: "cjs",
    bundle: true,
    external: [
      // Never re-bundle any library code to prevent accidentally duplicating
      "sietch",
      // It's critical that preact doesn't end up in the layout because that
      // breaks hooks/context when there are two copies (one in the layout
      // bundle and one in node_modules).
      "preact", "preact/hooks", "preact/jsx-runtime", "preact-render-to-string",
    ],
    outdir: site.cacheDir,
    plugins: [
      virtualModulesPlugin({
        filter: /^static:/,
        modules: {
          "static:islands": {
            loader: "tsx",
            resolveDir: site.rootDir,
            contents: createStaticSource(islands),
          },
        },
      }),
    ],
  });

  if (result.errors.length) {
    console.log("THERE WERE STATIC BUNDLE ERRORS")
    throw new EsbuildError(result.errors);
  }

  let layoutFile = join(site.cacheDir, "layout.js");
  let islandsFile = join(site.cacheDir, "islands.js");

  return { layoutFile, islandsFile };
}

async function compileClientBundles(site: Site) {
  let modules: Record<string, OnLoadResult> = {};
  let pages = site.pages.filter(hasHydratedIslands);

  for (let page of pages) {
    let islands = page.islands.filter(island => island.hydrate);
    if (islands.length === 0) continue;

    modules[`client:${page.id}`] = {
      loader: "tsx",
      resolveDir: join(site.pagesDir, page.dir),
      contents: createClientSource(islands),
    };
  }

  modules["client:layout"] = {
    contents: `import "${join(site.rootDir, "layout")}";`,
    loader: "ts",
    resolveDir: site.rootDir,
  };

  let result = await esbuild({
    entryPoints: Object.fromEntries(
      pages.filter(page => {
        return page.islands.some(island => island.hydrate);
      }).map(page => {
        return [page.id, `client:${page.id}`];
      }).concat([
        ["layout", "client:layout"]
      ]),
    ),
    platform: "browser",
    entryNames: site.env === "production" ? "[dir]/[name]-[hash]" : "[dir]/[name]",
    minify: site.env === "production",
    format: "esm",
    bundle: true,
    outdir: join(site.outDir, "assets"),
    splitting: true,
    metafile: true,
    plugins: [
      virtualModulesPlugin({
        filter: /^client:/,
        modules,
      }),
    ],
  });

  if (result.errors.length) {
    console.log("THERE WERE CLIENT BUNDLE ERRORS")
    throw new EsbuildError(result.errors);
  }

  let layoutEntryPoint = "virtual:client:layout";

  for (let page of site.pages) {
    let pageEntryPoint = `virtual:client:${page.id}`;

    for (let outfile in result.metafile.outputs) {
      let output = result.metafile.outputs[outfile];

      if (output.entryPoint === pageEntryPoint || output.entryPoint === layoutEntryPoint) {
        let url = "/" + relative(site.outDir, join(site.rootDir, outfile));
        page.scripts.push(url);

        if (output.cssBundle) {
          let url = "/" + relative(site.outDir, join(site.rootDir, output.cssBundle));
          page.styles.push(url);
        }
      }
    }
  }
}

function hasHydratedIslands(page: Page): boolean {
  return page.islands.some(island => island.hydrate);
}

function createStaticSource(islands: Island[]): string {
  return `
import { h } from "preact";
import { render } from "preact-render-to-string";
export let html = {};

${islands.map(island => `
import ${island.id} from "${island.src}";
html.${island.id} = render(h(${island.id}, ${JSON.stringify(island.props)}));
`).join("\n")}
`;
}

function createClientSource(islands: Island[]): string {
  return `
import { h, render, hydrate } from "preact";

${islands.map(island => `
import ${island.id} from "${island.src}";
${island.clientOnly ? "render" : "hydrate"}(
  h(${island.id}, ${JSON.stringify(island.props)}),
  document.querySelector("[data-island=${island.id}]"),
);
`).join("\n")}`;
}


/**
 * @internal
 */
export async function build(site: Site) {
  await readPages(site);
  await renderPages(site);
  await bundlePages(site);
  await writePages(site);
}
