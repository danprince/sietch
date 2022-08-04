{{ component "./say-hello" (props "name" "load") | hydrate }}
{{ component "./say-hello" (props "name" "idle") | hydrateOnIdle }}
{{ component "./say-hello" (props "name" "visible") | hydrateOnVisible }}
{{ component "./say-hello" (props "name" "only") | hydrate | clientOnly }}
