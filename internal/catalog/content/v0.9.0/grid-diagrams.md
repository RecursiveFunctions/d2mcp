# Grid diagrams

`grid-rows` and `grid-columns` arrange a container's children into a regular grid. Declaring both makes their source order determine whether rows or columns fill first. Width, height, and gap controls support dashboards, matrices, and table-like layouts.

```d2
matrix: Environments {
  grid-columns: 2
  grid-gap: 12
  dev: Development
  test: Test
  stage: Staging
  prod: Production
}
```

Use `grid-gap`, `vertical-gap`, and `horizontal-gap` to tune spacing. Grids can contain ordinary, grid, or sequence diagrams. Connections to a grid work normally; routing between its cells varies by engine. Dagre and ELK use direct center-to-center segments, while TALA applies its router.

Source: https://d2lang.com/tour/grid-diagrams/
