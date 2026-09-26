# Sequence diagrams

Set a container to `shape: sequence_diagram`. Declarations are ordered, actors share the sequence scope, and connections become messages. Nested objects on actors can represent activation spans or notes; unconnected containers form labeled groups.

```d2
login: Sign-in flow {
  shape: sequence_diagram
  user: User
  web: Web app
  api: API
  user -> web: submit credentials
  web -> api: authenticate
  api -> web: token
  web -> user: dashboard
}
```

Self-messages, groups, notes, styles, and custom actor shapes are supported. The whole sequence diagram is still an ordinary D2 object, so it can be nested or connected in a larger composition. D2 v0.9.0 keeps generated lifeline endpoint IDs stable across architectures.

Sources:
- https://d2lang.com/tour/sequence-diagrams/
- https://github.com/d2lang/d2/releases/tag/v0.9.0
