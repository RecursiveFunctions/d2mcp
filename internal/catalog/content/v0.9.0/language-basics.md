# Language basics

D2 describes a graph with keys, labels, maps, and connections. A bare key creates a rectangle; assigning text changes its displayed label. Nested maps create containers, and dotted paths target nested objects. Quoted keys preserve spaces and punctuation.

```d2
client: Mobile app
service: API {
  auth: Authentication
  data: Data store { shape: cylinder }
}
client -> service.auth: sign in
service.auth -> service.data: read
```

Semicolons may separate declarations on one line. Keys are identifiers, not labels: connections target `service.auth`, even if its label changes. `_` addresses the parent scope from inside a container.

Sources:
- https://d2lang.com/tour/shapes/
- https://d2lang.com/tour/containers/
- https://d2lang.com/tour/connections/
