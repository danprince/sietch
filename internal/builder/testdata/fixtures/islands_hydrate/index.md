{{ render "./say-hello" (props "name" "load") | clientOnLoad }}
{{ render "./say-hello" (props "name" "idle") | clientOnIdle }}
{{ render "./say-hello" (props "name" "visible") | clientOnVisible }}
{{ render "./say-hello" (props "name" "only") | clientOnLoad | clientOnly }}
