---
title: Islands
---

Use the `component` function in a template to render an ["island"](https://jasonformat.com/islands-architecture/).

```md
{{"{{ component \"./counter.tsx\" }} "}}
```

If the component accepts props, then these can be passed as an additional argument.

```md
{{"{{ component \"./counter.tsx\" (props \"count\" 1) }} "}}
```

_Note_: Go's template language doesn't support map/object literals, so you'll need to use the `props` function to construct one from a list of keys and values.

Unless you opt-in, components will only be rendered at build time. This means your page will include the HTML they rendered, but the JavaScript won't sent to browsers and the "island" won't be interactive.

## Hydrate
Use the `hydrate` function to make islands load their JavaScript and become interactive immediately.

```md
{{"{{ component \"./counter.tsx\" | hydrate }} "}}
```

### On Idle
Use the `hydrateOnIdle` function to make islands load their JavaScript and become interactive when the page calls `requestIdleCallback`.

```md
{{"{{ component \"./counter.tsx\" | hydrate }} "}}
```

### On Visible
Use the `hydrateOnVisible` function to make islands load their JavaScript and become interactive when they become visible onscreen.

```md
{{"{{ component \"./counter.tsx\" | hydrate }} "}}
```

### Client Only
Use the `clientOnly` function to prevent an island from rendering at build time. This can be useful for components that need to work with the DOM or other Web APIs that aren't available at build time.

```md
{{"{{ component \"./counter.tsx\" | hydrate | clientOnly }} "}}
```

## Frameworks
Sietch supports multiple frameworks for writing island components.

### Vanilla
If your component has a `.ts` or `.js` extension then Sietch will assume it's a "vanilla" island. You can also explicitly mark a file adding `.vanilla` before the extension (e.g. `counter.vanilla.tsx`). This tiny framework doesn't have any external libraries, which makes it a great choice for really simple components.

A vanilla component needs to export a `render` function to be rendered at build time and a `hydrate` function to be hydrated at runtime.

```ts
export function render(props): string {
  return `Hello, ${props.name}`;
}

export function hydrate(props, element: HTMLElement) {
  element.style.color = "red";
}
```

The `render` function takes the props that were passed when the component was rendered and must return a string of HTML.

The `hydrate` function receives the same set of props, and also the container element that the component was rendered into.

### Preact
If your component has a `.tsx` or `.jsx` extension, then Sietch will assume it's a "preact" island. You can also explicitly mark a file by adding `.preact` before the extension (e.g. `counter.preact.tsx`).

Preact islands need to expose their component with a default export.

```tsx
import { useState, useEffect } from "preact/hooks";

export default props => {
  let [style, setStyle] = useState({});

  useEffect(() => {
    setStyle({ color: "red" });
  }, [])

  <h1 style={style}>Hello, {props.name}</h1>
};
```

There's no need for the separation between render/hydrate with Preact, because the same function will be used when we call [hydrate](https://preactjs.com/guide/v10/api-reference/#hydrate) behind the scenes.

Like vanilla islands, the function receives the set of props that were passed when the component was rendered.

## Unsupported Frameworks
Compared to a tool like [Astro](https://docs.astro.build/en/guides/integrations-guide/), Sietch takes a fairly spartan approach to the frameworks it supports. Components that require custom compilers (like Vue and Svelte) won't ever be supported, because esbuild—the tool Sietch uses internally—[has no plans to support them either](https://esbuild.github.io/faq/#upcoming-roadmap).

What about React? React excels in complex web applications where the majority of the content is dynamic, and existing package ecosystem is a significant part of the work. By comparison Sietch wants to excel in pages that are 90% writing, with 10% interactive examples, and for that use case, Preact delivers a great user experience with a 'close enough' approximation and a fraction of the code.

<details>

<summary>It is still possible to write React components from vanilla islands, you'll just need a little bit more code.</summary>

```tsx
/** @jsxImportSource react */
import { useState } from "react";
import { hydrate as _hydrate } from "react-dom/client";

// https://github.com/anonyco/FastestSmallestTextEncoderDecoder/issues/18
import "fastestsmallesttextencoderdecoder/EncoderDecoderTogether.min";
import { renderToString } from "react-dom/server.browser";

let Counter = ({ count: init = 0 }) => {
  let [count, setCount] = useState(init);
  return <button onClick={() => setCount(count + 1)}>{count}</button>;
}

export function render(props) {
  return renderToString(<Counter {...props} />);
}

export function hydrate(props, element) {
  hydrate(<Counter {...props} />, element);
}
```

</details>

## Caveats
Because Sietch isn't a Node.js tool, there are some gotchas about the way it works that may trip you up if you're expecting everything to work the way it does in other platforms.

### TypeScript
TypeScript is an important part of many applications and Sietch supports TypeScript syntax for your islands.

However, the TypeScript ecosystem mostly lives in npm, which makes type checking third party code a challenge.

By default Sietch resolves imports to remote URLs behind the scenes. An import to `preact/hooks` becomes an import to `https://esm.sh/preact/hooks`. However, TypeScript has no idea this is happening an is instead expecting to find types for `preact/hooks` in `node_modules/preact/hooks` or `node_modules/@types/preact/hooks`.

In the future, I'd like to experiment with a model like Deno, where type definitions are 

### V8
Islands that rendered at build time are evaluated inside a V8 isolate, which has some important differences when compared to other server side rendering solutions that evaluate components inside node.js (Next, Remix, Gatsby).

- You don't have access to any of node's APIs. That means no file system, or network access, or environment variables.
- You don't have access to any web APIs. `window` and `document` will be `undefined` and it's up to guard any code that uses them.

### Esbuild
Node.js tools can't be used for transforming source code (including stuff like Babel, PostCSS, Uglify, etc).

Sietch comes with a few internal esbuild plugins, but can't be extended further with Go or JavaScript.
