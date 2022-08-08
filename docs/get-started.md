---
title: Get Started
nav: 1
---

Start by installing Sietch (currently this has a prerequisite on having Go installed).

```sh
# TODO: Update when there are cross platform binaries available
go install github.com/danprince/sietch
```

Create a new directory for your site and add an `index.md` file like the one below.

```md
---
title: Example Post
date: 2022-10-10
---

This is an example post.
```

Then run sietch to build your site.

```sh
sietch
```

During development you can serve your site locally with livereloading.

```sh
sietch --serve
```

## Islands & Interactivity

Add the following `counter.ts` file to the directory.

```ts
{{ embed "./snippets/counter.ts" }}
```

Then add the code to render it to your `index.md` page.

```go
{{"{{ component \"./counter\" (props \"count\" 0) }}"}}
```

Islands are static by default, which means they're rendered at build time and no JS is sent to the client.

That means that the button below won't actually do anything.

{{ component "./snippets/counter.ts" (props "count" 0) }}

We can fix that with the `hydrate` function. Opting-in to hydration means we'll ship the JS required to make the component interactive.

```go
{{"{{ component \"./counter\" (props \"count\" 0) | hydrate }}"}}
```

Let's see it in action.

{{ component "./snippets/counter.ts" (props "count" 0) | hydrate }}

Note that the button was still rendered at build time, so browsers with slow connections, or JavaScript disabled would still see a button as soon as they opened the page.

If you take a peek at the bundle we're sending to the client now, you'll notice that we aren't shipping the `render` function to the client, because it isn't used.

## Preact
You can also use [Preact](https://preactjs.com/) for your islands.

Now rename the `counter.ts` to `counter.tsx` and update the code to work with Preact.

```ts
{{ embed "./snippets/counter.tsx" }}
```

Now your island is powered by Preact.

{{ component "./snippets/counter.tsx" (props "count" 0) | hydrate }}

## What Next?
Check out the [islands reference](./reference/islands.md) for more information on using islands and writing components.
