# Composition

D2 can build multi-board documents with layers, scenarios, and steps. A layer is a separate board for another abstraction level. A scenario derives a board from a base board and applies changes. Steps present successive states and are useful for walkthroughs or animation.

```d2
app: Application
app -> db: stores

layers: {
  detail: {
    api: API
    worker: Worker
    api -> worker
  }
}
```

Board links let readers navigate composed SVG output. Use composition when one canvas would mix audiences or levels of detail; use ordinary containers when all objects should share one layout.

Sources:
- https://d2lang.com/tour/composition/
- https://d2lang.com/tour/layers/
- https://d2lang.com/tour/scenarios/
- https://d2lang.com/tour/steps/
