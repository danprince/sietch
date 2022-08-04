---
title: Markdown
---

Sietch supports the [CommonMark](https://commonmark.org/) specification with some extensions.

## Tables

```md
| foo | bar |
| --- | --- |
| baz | bim |
```

## Strikethrough

```md
~~Hi~~ Hello, world!
```

## Autolinks

```md
www.commonmark.org
```

## Tasklist

```md
- [ ] foo
- [x] bar
```

## Footnotes

```md
That's some text with a footnote.[^1]

[^1]: And that's the footnote.
```

## Heading Links
Headings are given an automatic ID and wrapped in a link

```md
# Hello

Link to [](#hello).
```

## External Links
External links automatically get `target="_blank"` and `rel="noreferrer noopener"` attributes.

```md
[Opens in the current tab](./islands.html)
[Opens in a new tab](http://danthedev.com)
```

## Page Links
Links to other markdown files are automatically translated to the equivalent HTML.

```md
[Opens ./islands.html](./islands.md)
```

## Code Highlighting
Fenced code blocks support a [Prism style syntax](https://prismjs.com/plugins/line-highlight/) for line range highlights (e.g. `js/2-4`)

    ```ts/2-4
    export function onIdle(): Promise<void> {
      return new Promise(resolve => {
        requestIdleCallback(() => resolve());
      })
    }
    ```
