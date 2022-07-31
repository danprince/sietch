---
title: Playground
index: true
---

I only render at the server.
{{ render "./_counter.preact.tsx" (props "count" 1) }}

I render at the server and the client
{{ render "./_counter.preact.tsx" (props "count" 2) | clientLoad }}

I only render at the client
{{ render "./_counter.preact.tsx" (props "count" 3) | clientOnly }}
