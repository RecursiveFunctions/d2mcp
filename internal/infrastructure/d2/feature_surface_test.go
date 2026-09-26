package d2

import (
	"context"
	"testing"
)

func TestD2V090LanguageSurfaceValidates(t *testing.T) {
	repo := NewD2Repository().(*D2Repository)
	fixtures := map[string]string{
		"containers-connections-arrowheads": `cloud: Cloud {
  api: API
  db: Database {shape: cylinder}
  api -> db: query {target-arrowhead.shape: diamond}
}
client <-> cloud.api: request`,
		"rich-text-code-math": `docs: |md
# Runbook
Use **D2** diagrams.
|
code: |go
func main() {}
|
formula: |latex
x^2 + y^2
|`,
		"sql-table": `users: Users {
  shape: sql_table
  id: int {constraint: primary_key}
  team_id: int {constraint: foreign_key}
}
teams: Teams {
  shape: sql_table
  id: int {constraint: primary_key}
}
users.team_id -> teams.id`,
		"uml-class": `account: Account {
  shape: class
  -balance: decimal
  +deposit(amount decimal): void
}`,
		"sequence": `login: Sign-in flow {
  shape: sequence_diagram
  user: User
  web: Web app
  api: API
  user -> web: submit credentials
  web -> api: authenticate
  api -> web: token
  web -> user: dashboard
}`,
		"grid": `matrix: Environments {
  grid-columns: 2
  grid-gap: 12
  dev: Development
  test: Test
  stage: Staging
  prod: Production
}`,
		"classes-styles": `classes: {
  service: {
    shape: rectangle
    style.fill: "#e8f1ff"
    style.border-radius: 8
  }
}
api: API {class: service}
worker: Worker {class: service}
api -> worker`,
		"variables-globs-config": `vars: {
  primary: "#4466aa"
  d2-config: {
    layout-engine: dagre
    theme-id: 0
  }
}
a
b
a -> b
*.style.fill: ${primary}
**.style.stroke-width: 2`,
		"composition": `app: Application
app -> db: stores
layers: {
  detail: {
    api: API
    worker: Worker
    api -> worker
  }
}
scenarios: {
  failure: {
    app.style.fill: red
  }
}
steps: {
  one: {
    app.label: Started
  }
}`,
	}

	for name, source := range fixtures {
		t.Run(name, func(t *testing.T) {
			result, err := repo.Validate(context.Background(), source)
			if err != nil {
				t.Fatalf("Validate() error = %v", err)
			}
			if !result.Valid {
				t.Fatalf("D2 v0.9 fixture is invalid: %+v", result.Diagnostics)
			}
		})
	}
}
