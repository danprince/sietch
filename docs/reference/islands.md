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

## HTTP Imports
In addition to importing from relative files and from `node_modules`, components can also import directly from HTTPS urls.

```md
import { moo } from "https://esm.sh/cowsayjs@1.0.7";

export function render({ message }: { message: string }): string {
  return moo(message);
}
```

This is a useful way to consume dependencies if you want to use Sietch without installing nodejs or npm.

HTTP imports will be downloaded and cached at build time, then bundled when your site is built. This means that they won't actually be used at runtime.

At some point, Sietch will support some form of user configured import maps to make managing HTTP imports simpler.

## Frameworks
### Vanilla
The default islands framework is a simple format where each islands file can export two functions.

### Preact

### Preact Remote

## TypeScript


## Caveats

### V8

### ESBuild
