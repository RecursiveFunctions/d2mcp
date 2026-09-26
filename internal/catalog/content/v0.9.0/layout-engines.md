# Layout engines

D2 v0.9.0 bundles Dagre, ELK, and the now-open-source TALA engine. Dagre is the default fast hierarchical layout; ELK offers mature hierarchical routing; TALA targets software architecture and supports the broadest D2-specific positioning controls.

Select an engine with `--layout`, `D2_LAYOUT`, or `vars.d2-config.layout-engine`. TALA no longer needs a plugin or license key. Its search effort can be tuned with `--tala-seeds`, `D2_TALA_SEEDS`, or `vars.d2-config.data.tala-seeds`.

```d2
vars: {
  d2-config: {
    layout-engine: tala
    data.tala-seeds: 8
  }
}
direction: right
client -> api -> database
```

Layout support differs: TALA alone supports per-container directions and fixed `top`/`left`; container dimensions work in TALA and ELK.

Sources:
- https://d2lang.com/tour/layouts/
- https://github.com/d2lang/d2/releases/tag/v0.9.0
