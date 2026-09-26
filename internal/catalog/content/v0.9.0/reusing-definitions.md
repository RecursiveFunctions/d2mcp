# Reusing definitions

Classes collect reusable object or connection attributes. Apply one class by name or multiple classes as an ordered array; explicit attributes on the target win. Variables provide reusable scalar or map values. Imports bring in another `.d2` file as a map, spread its members into the current map, or select a nested object.

```d2
classes: {
  service: {
    shape: rectangle
    style.fill: "#e8f1ff"
    style.border-radius: 8
  }
}
api: API { class: service }
worker: Worker { class: service }
api -> worker
```

```d2
shared: @components
...@theme
```

Imports resolve relative to the importing file. D2 v0.9.0 also detects class-reference cycles rather than recursing indefinitely.

Sources:
- https://d2lang.com/tour/classes/
- https://d2lang.com/tour/imports/
- https://github.com/d2lang/d2/releases/tag/v0.9.0
