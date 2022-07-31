---
title: Theme
nav: 3
---

Sietch comes with a default theme (you're looking at it), but it's easy to customise.

Every page is rendered through a `_template.html` file in the top level directory of your site.

## Overview
Go's [templating language](https://pkg.go.dev/text/template) is used. Here's a quick overview of the basic features.

```html
<!-- Render the page contents -->
{{"{{ .Contents }}"}}

<!-- Render properties from the current page's front matter -->
{{"{{ .Data.title }}"}}

<!-- Render conditionally -->
{{"{{ if .Data.title }}"}}
{{"  <h1>{{ .Data.title }}</h1>"}}
{{"{{ end }}"}}
```

Reading the `template.html` in the [repository](https://github.com/danprince/sietch) is a good place to start.

Hugo also uses the same language, and [their documentation](https://gohugo.io/templates/introduction/) can help here too. Just remember that you won't have access to the same functions and variables.

Finally, the [Go documentation for `text/template`](https://pkg.go.dev/text/template) is the best place to go to really understand the language.

## Variables
The variables you have access to in these templates come from the page that is currently being rendered.

### `.Contents`
The contents of the page (_after_ it has been converted from markdown to HTML). Unless you like empty pages, you need to render this somewhere in your template.

```html
<main>
  {{"{{ .Contents }}"}}
</main>
```

### `.Data`
The front matter data from the page.

```html
<!-- Render the title -->
{{"{{ .Data.title }}"}}

<!-- Render a cover image -->
{{"<img src=\"{{ .Data.cover }}\" />"}}
```

### `.Url`
The page's URL.

```html
{{"<a href=\"{{ .Url }}\">Link to the current page</a>"}}
```

### `.Date`
The date for the current page. This is an object that needs to be formatted if you want to render it.

```html
<time>{{"{{ .Date.Format \"Jan 2, 2006\" }}"}}</time>
```

For more information about Go's date formatting string, [Hugo has a great guide](https://gohugo.io/functions/format/#gos-layout-string).

## Functions
In addition to the page variables, there are also some functions for doing slightly more fancy stuff.

### `nav`
Returns the list of pages which have `nav: true` in their frontmatter. This can be useful for re-creating the navbar from the default theme.

```html
{{- `
{{ range nav }}
  <a href="{{- .Url -}}">
    {{- .Data.title -}}
  </a>
{{ end }}
` -}}
```

### `index`
Returns the list of pages that are in the current directory.

```html
{{- `
{{ range index }}
  <a href="{{- .Url -}}">
    {{- .Data.title -}}
  </a>
{{ end }}
` -}}
```

This list _excludes_ any `index.md` files in the current directory, and _includes_ any `index.md` files from subdirectories.

### `include`
Now you're playing with fire. The `include` function allows you to include another file directly inside your template.

```html
<header>
  {{"{{ include \"_nav.html\" }}"}}
</header>
```

Using `include` is an easy way to overcomplicate your template, so don't abuse it.

## Errors
Go's templating language isn't the most beginner friendly, but it does a nice job at catching potential errors when compiling your templates.

If Sietch's build fails, we'll show the error in your browser, so that you don't have to go back to the terminal to figure out what went wrong.

<pre style="overflow-x: auto;font-family: Consolas,Menlo,Monaco,monospace;border-radius:8px;border: solid 3px #e41010;background:white;padding: 16px;"><span style="color: #e41010; font-weight: bold">error:</span> template evaluation error

<span style="font-weight: bold">./theme.md:118</span>
<span style="color: #adadad">116</span> If Sietch's build fails, we'll show the error in your browser, so that you don't have to go back to the terminal to figure out what went wrong.
<span style="color: #adadad">117</span> 
<span style="font-weight: bold">118</span> {{"{{ nav 2 }}"}}
    <span style="color: #e41010; font-weight: bold">^^^^^^^^^^^</span>
    <span style="color: #e41010; font-weight: bold">wrong number of args for nav: want 0 got 1</span>
<span style="color: #adadad">119</span> 
<span style="color: #adadad">120</span> 
</pre>
