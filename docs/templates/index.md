---
title: Templates
---

To keep things simple, every page is rendered through a `_template.html` file at the root of your project.

Go's [templating language](https://pkg.go.dev/text/template) is used to manage to the logic inside. You'll already be familiar with it if you have used Hugo.

Here are some common examples.

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

See the `template.html` from the repository for a reference.

### Variables
In templates you have access to the current page, which looks like this:

```go
type Page struct {
  // The url for the page
  Url      string
  // Any data in the page's frontmatter
  Data     map[string]any
  // The rendered html content of the page
  Contents string
}
```

### Functions

#### `date`

### Errors
When you're in the flow of writing, you don't want to go back to the terminal to figure out why things aren't updating.

Sietch fails fast and also shows the error in your browser too. Here's an example:

<pre style="background: white; overflow-x: auto;font-family: Consolas,Menlo,Monaco,monospace;margin: 32px;border: 0;"><span style="color: #e41010; font-weight: bold">error:</span> template evaluation error

<span style="font-weight: bold">./templates.md:42</span>
<span style="color: #adadad"> 41</span> 
<span style="font-weight: bold"> 42</span> #### `{{"{{ date }}"}}`
    <span style="color: #e41010; font-weight: bold">^^^^^^^^^^^^^^^^^</span>
    <span style="color: #e41010; font-weight: bold">wrong number of args for date: want 3 got 0</span>
<span style="color: #adadad"> 43</span> 
</pre>
