# Connections

D2 supports undirected (`--`), forward (`->`), reverse (`<-`), and bidirectional (`<->`) connections. Referencing an unknown key creates that shape. Repeating a connection creates another edge rather than replacing the first, and chains provide compact paths.

```d2
browser -> gateway: HTTPS
gateway -> api -> database
api <-> queue: events
```

Connections can have styles, classes, labels, and endpoint arrowhead objects. A repeated edge can be targeted by its zero-based index for later customization.

```d2
api -> database
api -> database
(api -> database)[1].style.stroke-dash: 4
```

Arrowhead shapes include triangle, arrow, diamond, circle, box, cross, and crow's-foot variants.

Source: https://d2lang.com/tour/connections/
