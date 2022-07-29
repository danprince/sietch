---
nav:
  About: /
  Templates: /templates/
index: true
---

<div style="text-align: center">
  <h1>Sietch</h1>
  <small>/siːɛtʃ/</small>
  <p>An uncompromisingly simple static site generator.</p>
</div>

<div style="display: flex; justify-content: center; align-items: center; gap: 64px">
<div>

```
index.md
tabr.md
spice.png
_harkonnens
.gitignore
```

</div>
<div>
→
</div>
<div>

```
_site/
  index.html
  tabr.html
  spice.png

```

</div>
</div>

__Build blogs, not spaceships__. No dynamic pages. No flexible data sources. No query languages. No taxonomies. No plugins.

__Markdown goes in, HTML comes out__. Your site has a 1:1 mapping with your file system. Group pages into directories for a content hierarchy.

__Make it your own__. Use a single [`_template.html`](/templates.html) file without going down a rabbit hole of partials and layout inheritance.

## Get Started
Once installed, create an `index.md` file in an empty directory.

```md
---
title: My Site
date: 2020-07-29
---

Hello from sietch.
```

The default theme will automatically render the `title` and `date` fields when they are present.

```sh
# build the site in your cwd
sietch

# serve on http://localhost:8000 and rebuild on refresh
sietch --serve
```

## Tips
- Files and directories starting with `.` or `_` will not be included in your site.
- Pages can automatically render a list of their siblings below the contents. This includes `index.md` files in any subdirectories. Set `index: true` in the frontmatter of any file to enable.

## Frequently Asked Questions
- __How do I extend Sietch?__ You don't. You can change the way it looks and renders with a [custom template](templates.html), but you can't change the way it works.
- __What flavour of markdown is supported?__ Sietch supports [CommonMark](https://commonmark.org/) with the [GFM](https://github.github.com/gfm/) and [Footnotes](https://michelf.ca/projects/php-markdown/extra/#footnotes) extensions.
- __What does Sietch mean?__ A sietch is a home in an otherwise inhospitable desert landscape. Borrowed from Dune by Frank Herbert.
