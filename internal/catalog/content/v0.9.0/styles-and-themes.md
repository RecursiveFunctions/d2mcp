# Styles and themes

Set visual attributes under `style`, directly or through classes. Supported controls include opacity, stroke, fill, patterns, stroke width and dash, border radius, shadow, 3D, multiple, double border, font size and color, animation, emphasis, and text transform. Colors accept CSS names, hex values, and supported gradients.

```d2
classes: {
  warning: {
    style.fill: "#fff3cd"
    style.stroke: "#b58105"
    style.bold: true
  }
}
timeout: Request timed out { class: warning }
```

Root styles control diagram background and frame. Themes supply coordinated defaults while explicit styles override them. Some effects depend on shape or layout support; for example, rounded connection corners matter only when a routed edge has corners.

Sources:
- https://d2lang.com/tour/style/
- https://d2lang.com/tour/themes/
