---
title: Config
---

Sietch supports a small number of configuration options with a `.sietch.json` file at the root of your site directory.

## `PagesDir`
_Default: `.`_

This is the directory where Sietch will search for markdown files to turn into pages. By default it is the root of the site directory, but you use any relative directory.

For example, if you set `PagesDir` to `"./posts"` then only pages in the `posts` directory will be built. This doesn't affect where Sietch looks for config files, templates, or public files.

## `DateFormat`
_Default: `2006-1-2`_

Sietch only understands dates in the format `2022-1-20` (20th January 2022) by default. Set an alternate format here to use other date formats.

The value must be a layout that [Go's `time` package](https://pkg.go.dev/time) can parse. That means it must use the reference time of `01/02 03:04:05PM '06 -0700`.

Here are [some examples](https://pkg.go.dev/time#pkg-constants) of other valid date formats.

## `SyntaxColor`
_Default: [`algol_nu`](https://xyproto.github.io/splash/docs/longer/algol_nu.html)_

Set the syntax highlighting theme to one of the options from [Chroma's styles](https://xyproto.github.io/splash/docs/all.html).

If you would prefer to style your code with CSS, use `"css"` as the value here instead.

## `Npm`
_Default: `false`_

Setting `Npm` to `true` will ensure that the bundler tries to resolve files from your local `node_modules` directory, rather than from esm.sh.

## `ImportMap`
_Default: `{}`_

The import map is used for custom module resolutions. Usually that would be pinning a module to a specific version.

```json
{
  "ImportMap": {
    "preact": "preact@10.10.1",
    "preact/hooks": "preact@10.10.1/hooks",
    "preact/jsx-runtime": "preact@10.10.1/jsx-runtime"
  }
}
```

Or aliasing one module to another.

```json
{
  "ImportMap": {
    "react": "preact/compat",
    "react/jsx-runtime": "preact/compat/jsx-runtime"
  }
}
```

It can also be used for setting any of the other flags that [esm.sh](https://esm.sh/) uses.

```json
{
  "preact": "preact?bundle",
  "preact": "preact?dev",
  "preact-render-to-string": "preact-render-to-string?external=preact",
  "preact": "preact?pin=v90"
}
```

_Note:_ Import maps only work with complete matches. Mapping `preact` to `preact@10` won't automatically catch imports for `preact/hooks` or other subpackage exports.
