<header style="padding: 64px; text-align: center">
  <h1>Sietch</h1>
  An uncompromisingly simple static site generator.
</header>


1. ### Markdown → HTML
    Every markdown file in your directory becomes an HTML file in your site.
2. ### Writing, not rocket science
    No dynamic pages. No flexible data sources. No query languages. No taxonomies. No plugins.

### Get Started
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

## Tips & Tricks
- #### Ignore Files
    Files and directories starting with `.` or `_` will not be included in your site.
- #### Index Pages
    Pages can automatically render a list of links to adjacent pages. Set `index: true` in the frontmatter of any page to enable.

## Frequently Asked Questions
- #### How do I extend Sietch?
    You don't. You can change the way it looks and renders with a [custom template](templates.html), but you can't change the way it works.
- #### What flavour of markdown is supported?
    Sietch supports [CommonMark](https://commonmark.org/) with the [GFM](https://github.github.com/gfm/) and [Footnotes](https://michelf.ca/projects/php-markdown/extra/#footnotes) extensions.
- #### What does Sietch mean?
    A sietch is a home in an otherwise inhospitable desert landscape (borrowed from [Dune](https://en.wikipedia.org/wiki/Dune_(novel))).
- #### Have these questions really been asked frequently?
    No.
