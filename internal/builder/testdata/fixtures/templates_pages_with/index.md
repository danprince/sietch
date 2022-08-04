---
nav: true
title: Home
---
<nav>
  {{ range pagesWith "nav" }}
  <a href="{{ .Url }}">{{ .Data.title }}</a>
  {{ end }}
</nav>
