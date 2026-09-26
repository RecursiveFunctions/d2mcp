# Text, code, and math

Block strings attach rich labels to shapes. `md` renders Markdown, language tags request syntax highlighting, and `latex` or `tex` renders MathJax notation. Unknown code languages fall back to plain text. `shape: text` creates unframed text.

```d2
note: |md
  ## Deployment
  Run only after tests pass.
|

formula: |latex
  E = mc^2
|

sample: |go
  if err != nil {
    return err
  }
|
```

D2 supports Unicode labels, including non-Latin scripts and emoji. In v0.9.0, Markdown labels render as native SVG instead of HTML `foreignObject`, improving portability and output size.

Sources:
- https://d2lang.com/tour/text/
- https://github.com/d2lang/d2/releases/tag/v0.9.0
