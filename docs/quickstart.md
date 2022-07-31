---
title: Quickstart
nav: 2
---

### Install
Install the binary with `go install` ([gobinaries.com](https://gobinaries.com) alternative coming soon).

```sh
go install github.com/danprince/sietch
```

### Setup

Then create a directory for your site and add an `index.md` file to it.

```md
---
title: My Site
date: 2020-07-29
---

Hello from sietch.
```

The default theme will automatically render the `title` and `date` fields when they are present.

### Run
Run sietch with the serve flag to start a server on http://localhost:8000 that will automatically rebuild your site after changes.

```sh
sietch --serve
```

For now, there's no mechanism to inject JavaScript into your HTML files, so there are live reloads after changes. Instead, you'll need to refresh your page manually like a caveman. 

### Build
Run sietch without any flags to clean your output directory and build your site.

```sh
sietch
```

Now you can deploy the `_site` directory to the web.
