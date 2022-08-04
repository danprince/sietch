---
title: Home
nav: 0
---
<nav>
  {{ range pagesWith "nav" | sortBy "nav" }}
  <a href="{{ .Url }}">{{ .Data.title }}</a>
  {{ end }}
</nav>
