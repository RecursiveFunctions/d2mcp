# Export capabilities in v0.9.0

D2 v0.9.0 can render SVG, PNG, GIF, PDF, PPTX, and terminal-friendly ASCII through its built-in renderers. This release removes the earlier 67-megapixel PNG ceiling, lowers PNG memory use, and honors `D2_TIMEOUT` during PNG and GIF rendering.

```text
d2 architecture.d2 architecture.svg
d2 architecture.d2 architecture.png
d2 --animate-interval 1200 states.d2 states.gif
d2 deck.d2 deck.pptx
d2 architecture.d2 architecture.txt
```

The v0.9.0 renderer substantially reduces repeated layout and rendering work. Official release measurements report roughly 10× faster SVG rendering across the project's end-to-end corpus and smaller SVG output, while noting that results vary by diagram. Markdown labels are emitted as native SVG.

Sources:
- https://github.com/d2lang/d2/releases/tag/v0.9.0
- https://d2lang.com/tour/intro/
