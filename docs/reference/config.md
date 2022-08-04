---
title: Config
---

Sietch supports a small number of configuration options with a `.sietch.json` file at the root of your site directory.

## `DateFormat`
Sietch only understands dates in the format `2022-1-20` (20th January 2022) by default. Set an alternate format here to use other date formats.

The value must be a layout that [Go's `time` package](https://pkg.go.dev/time) can parse. That means it must use the reference time of `01/02 03:04:05PM '06 -0700`.

Here are [some examples](https://pkg.go.dev/time#pkg-constants) of other valid date formats.

## `SyntaxColor`
Set the syntax highlighting theme to one of the options from [Chroma's styles](https://xyproto.github.io/splash/docs/all.html).

If you would prefer to style your code with CSS, use `"css"` as the value here instead.

## `Framework`
The framework to use for islands. Can be one of:

- `vanilla` (plain JS components)
- `preact` (Preact preinstalled in `node_modules`)
- `preact-remote` (Preact installed from cdn)

See the [islands reference](./islands.html) for details about the frameworks.
