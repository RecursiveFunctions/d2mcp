# Structured diagrams

D2 has dedicated shapes for common notation. `sql_table` models fields, types, and constraints for ERDs. `class` models UML fields and methods, including public (`+`), private (`-`), and protected (`#`) visibility.

```d2
users: Users {
  shape: sql_table
  id: int { constraint: primary_key }
  team_id: int { constraint: foreign_key }
}
teams: Teams {
  shape: sql_table
  id: int { constraint: primary_key }
}
users.team_id -> teams.id
```

```d2
account: Account {
  shape: class
  -balance: decimal
  +deposit(amount decimal): void
}
```

These remain normal D2 objects: they may be nested, connected, styled, and assigned classes.

Sources:
- https://d2lang.com/tour/sql-tables/
- https://d2lang.com/tour/uml-classes/
