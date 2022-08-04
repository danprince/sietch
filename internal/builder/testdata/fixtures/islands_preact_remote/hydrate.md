{{ component "./component" (props "count" 1) | hydrate }}
{{ component "./component" (props "count" 2) | hydrateOnIdle }}
{{ component "./component" (props "count" 3) | hydrateOnVisible }}
{{ component "./component" (props "count" 4) | hydrate | clientOnly }}
