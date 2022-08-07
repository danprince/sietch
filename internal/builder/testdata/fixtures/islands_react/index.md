{{ component "./counter.vanilla" (props "count" 1) }}
{{ component "./counter.vanilla" (props "count" 2) | hydrate }}
{{ component "./counter.vanilla" (props "count" 3) | hydrateOnIdle }}
{{ component "./counter.vanilla" (props "count" 4) | hydrateOnVisible }}
