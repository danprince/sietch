---
title: Pages
---

Sietch turns `.md` files into corresponding `.html` files.

## Ignored Files
Markdown files and directories that start with `_` or `.` are ignored.

## Public Dir
If your site has a `public` directory, then everything inside will be copied into `_site` recursively.

## Nav Pages
_Default template only_

Add `nav: true` to the page's frontmatter to add it to the navigation links. The page's `title` will be used as the text.

Use numbers instead to customise the order of those pages. For example a page with `nav: 0` will always show up before a page with `nav: 4`. 

### Index Pages
_Default template only_

Add `index: true` to the front matter of any page to render a list of links to adjacent pages below the page's content.

Adjacent `index.md` files won't show up in this list, but `index.md` files from subdirectories will.

This is a great way to render a list of all posts from your homepage.
