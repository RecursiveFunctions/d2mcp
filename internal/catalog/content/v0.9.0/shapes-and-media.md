# Shapes and media

Shapes default to `rectangle`. Built-in forms include square, page, parallelogram, document, cylinder, queue, package, step, callout, stored_data, person, diamond, oval, circle, hexagon, cloud, and c4-person. Some shapes retain a 1:1 aspect ratio.

```d2
user: Customer { shape: person }
db: Orders { shape: cylinder }
user -> db: query
```

Set `icon` to an HTTPS URL or local image path. Icon placement adapts to labels and containers. Use `shape: image` when the image itself should be the standalone shape.

```d2
logo: {
  shape: image
  icon: https://example.com/logo.svg
}
```

Remote images embedded by v0.9.0 can be decoded when served with gzip, Brotli, or deflate compression.

Sources:
- https://d2lang.com/tour/shapes/
- https://d2lang.com/tour/icons/
- https://github.com/d2lang/d2/releases/tag/v0.9.0
