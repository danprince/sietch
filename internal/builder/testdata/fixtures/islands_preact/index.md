{{ component "./counter" (props "count" 1) }}
{{ component "./counter" (props "count" 2) | hydrate }}
{{ component "./counter" (props "count" 3) | hydrateOnIdle }}
{{ component "./counter" (props "count" 4) | hydrateOnVisible }}
