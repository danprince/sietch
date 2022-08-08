---
title: Sietch
nav: 0
---

Sietch is a markdown powered static site generator that can render and bundle interactive TypeScript components at build time, without Node.js or npm.

It's designed for people who want to write posts with small amounts of interactive content as examples or demos.

Here are a few reasons you might pick Sietch.

1. ### You want to write, not build a spaceship
    No plugins. No dynamic data sources. No taxonomies. No partials. No data cascades. No generated pages.
2. ### It works out of the box
    Sietch is a standalone binary with everything you need to write markdown, and render components written in TypeScript or JavaScript.
3. ### Simplicity + consistency > flexibility
    Every single page is rendered through one customisable template file. There are no per-page layouts, partials, or inheritance structures.

## Why Not Sietch?
There are also some reasons why Sietch might not be a good fit.
- It's an early stage experimental project
- You want lots of types of pages, each with a distinct layout and theme.
- You want to generate your pages from a remote data source.
- You want to use a compiled language for components (Vue, Svelte, etc)
- You want to customize the way pages render markdown
- You want to build a web app, with a focus on dynamic content
- You need Node.js APIs for your components (file system access, network, etc)

If you made it this far, that's probably a good sign. Let's have a look at some

## Islands
Sietch's component model is based on the [islands architecture](https://jasonformat.com/islands-architecture/) and takes a significant amount of inspiration from [Astro](https://astro.build/) and [Slinkity](https://slinkity.dev/).

That means you can render a TypeScript/JavaScript component into your markdown, and it will be evaluated at build time.

```md
# Static Counter
{{"{{ component \"./counter.tsx\" (props \"count\" 10) }}"}}
```

By default these components aren't 'hydrated' which means that no JavaScript will be sent to the client side along with the markup.

However, you can hydrate individual components at runtime, so that they'll become interactive in browsers.

```md
# Interactive Counter
{{"{{ component \"./counter.tsx\" (props \"count\" 10) | hydrate }}"}}
```

You can also defer loading the JavaScript until the page becomes idle, or the component becomes visible.

```md
# Lazy Islands
{{"{{ component \"./counter.tsx\" (props \"count\" 10) | hydrateOnIdle }}"}}

{{"{{ component \"./counter.tsx\" (props \"count\" 10) | hydrateOnVisible }}"}}
```

Sietch supports both [Preact](https://preactjs.com/) and [Vanilla](http://localhost:8000/reference/islands.html#vanilla) components. See the [islands reference](./reference/islands.md) docs for more information.

## How does it work?
[V8](https://v8.dev/), [esbuild](https://esbuild.github.io/), and [esm.sh](https://esm.sh/) do most of the heavy lifting behind the scenes.

Sietch keeps record of all the islands it has seen whilst it builds pages, then uses them to create a static rendering bundle, that generates a string of HTML for each island.

If the bundle includes non-local dependencies, they will be resolved, downloaded and cached from [esm.sh](https://esm.sh).

Then the bundle is evaluated inside V8 (a sandbox which has no access to the file system or network) and the resulting HTML is used to fill in the blanks.

Next Sietch creates a bundle for each page with hydrated components. These bundles have code splitting applied, so that you won't end up serving two copies of one common library for two separate pages.

Because your code is run through esbuild, modern JavaScript & TypeScript syntax is also supported, including JSX/TSX, but you'll need to install TypeScript separately if you actually want to type check your codebase.

## Roadmap
Sietch is already fairly close to feature complete. Here are some of the things that I plan to add still.

- Support type definitions downloaded from esm.sh using the `x-typescript-types` header.
- Basic pagination (next/prev links at least)

Here are the things that I'm still considering for scope.
- JS/CSS bundles without islands
- Search / indexing

Here are some examples of features that are out of scope.
- Plugin system for customising builds
- Build time data fetching
- Dynamic page generation
- Multi-file templates/layout inheritance
- Page specific layouts
- Support for JavaScript based compilers (e.g. Vue, Svelte)
- Automatically including polyfills
- SSR using Node rather than V8

## Known Issues
- You can't use template directives in the front matter section of your pages.
- CSS imported in non-hydrated bundles won't be included in the page.
