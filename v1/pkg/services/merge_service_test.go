package services

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"helm-charts-migrator/v1/pkg/config"
	"helm-charts-migrator/v1/pkg/yaml"
)

func TestMergeService_MergeWithComments(t *testing.T) {
	tests := []struct {
		name         string
		baseYAML     string
		overrideYAML string
		wantAdded    []string
		wantUpdated  []string
		wantErr      bool
		validate     func(t *testing.T, merged []byte)
	}{
		{
			name: "simple merge with new keys",
			baseYAML: `
key1: value1
key2: value2
nested:
  a: 1
  b: 2
`,
			overrideYAML: `
key2: updated_value
key3: value3
nested:
  b: 3
  c: 4
`,
			wantAdded:   []string{"key3", "nested.c"},
			wantUpdated: []string{"key2", "nested.b"},
			wantErr:     false,
			validate: func(t *testing.T, merged []byte) {
				// Parse merged result
				var result map[string]interface{}
				err := yaml.Unmarshal(merged, &result)
				require.NoError(t, err)
				
				assert.Equal(t, "value1", result["key1"])
				assert.Equal(t, "updated_value", result["key2"])
				assert.Equal(t, "value3", result["key3"])
				
				nested := result["nested"].(map[string]interface{})
				assert.Equal(t, float64(1), nested["a"])
				assert.Equal(t, float64(3), nested["b"])
				assert.Equal(t, float64(4), nested["c"])
			},
		},
		{
			name: "deep nested merge",
			baseYAML: `
level1:
  level2:
    level3:
      key: base_value
      list:
        - item1
        - item2
`,
			overrideYAML: `
level1:
  level2:
    level3:
      key: override_value
      newkey: newvalue
      list:
        - item3
        - item4
        - item5
`,
			wantAdded:   []string{"level1.level2.level3.newkey"},
			wantUpdated: []string{"level1.level2.level3.key", "level1.level2.level3.list"},
			wantErr:     false,
		},
		{
			name:         "invalid base yaml",
			baseYAML:     "not: valid: yaml: [}",
			overrideYAML: "key: value",
			wantErr:      true,
		},
		{
			name:         "invalid override yaml",
			baseYAML:     "key: value",
			overrideYAML: "not: valid: yaml: [}",
			wantErr:      true,
		},
		{
			name: "empty override",
			baseYAML: `
key1: value1
key2: value2
`,
			overrideYAML: "",
			wantAdded:    []string{},
			wantUpdated:  []string{},
			wantErr:      false,
		},
	}

	cfg := &config.Config{}
	svc := NewMergeService(cfg)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			merged, report, err := svc.MergeWithComments([]byte(tt.baseYAML), []byte(tt.overrideYAML))
			
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			
			require.NoError(t, err)
			require.NotNil(t, merged)
			require.NotNil(t, report)
			
			// Check report
			assert.ElementsMatch(t, tt.wantAdded, report.AddedKeys)
			assert.ElementsMatch(t, tt.wantUpdated, report.UpdatedKeys)
			
			// Validate merged content
			if tt.validate != nil {
				tt.validate(t, merged)
			}
		})
	}
}

func TestMergeService_TrackChanges(t *testing.T) {
	tests := []struct {
		name        string
		before      map[string]interface{}
		after       map[string]interface{}
		wantAdded   int
		wantUpdated int
		wantDeleted int
	}{
		{
			name: "track all change types",
			before: map[string]interface{}{
				"keep":   "same",
				"update": "old",
				"delete": "gone",
				"nested": map[string]interface{}{
					"keep":   "same",
					"update": "old",
					"delete": "gone",
				},
			},
			after: map[string]interface{}{
				"keep":   "same",
				"update": "new",
				"add":    "new",
				"nested": map[string]interface{}{
					"keep":   "same",
					"update": "new",
					"add":    "new",
				},
			},
			wantAdded:   2, // add, nested.add
			wantUpdated: 2, // update, nested.update
			wantDeleted: 2, // delete, nested.delete
		},
		{
			name:        "no changes",
			before:      map[string]interface{}{"key": "value"},
			after:       map[string]interface{}{"key": "value"},
			wantAdded:   0,
			wantUpdated: 0,
			wantDeleted: 0,
		},
		{
			name:        "all new",
			before:      map[string]interface{}{},
			after:       map[string]interface{}{"key1": "value1", "key2": "value2"},
			wantAdded:   2,
			wantUpdated: 0,
			wantDeleted: 0,
		},
		{
			name:        "all deleted",
			before:      map[string]interface{}{"key1": "value1", "key2": "value2"},
			after:       map[string]interface{}{},
			wantAdded:   0,
			wantUpdated: 0,
			wantDeleted: 2,
		},
	}

	cfg := &config.Config{}
	svc := NewMergeService(cfg)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			changes := svc.TrackChanges(tt.before, tt.after)
			
			require.NotNil(t, changes)
			assert.Len(t, changes.Added, tt.wantAdded)
			assert.Len(t, changes.Updated, tt.wantUpdated)
			assert.Len(t, changes.Deleted, tt.wantDeleted)
		})
	}
}

func TestMergeService_ComplexNestedMerge(t *testing.T) {
	baseYAML := `
application:
  name: myapp
  version: 1.0.0
  config:
    database:
      host: localhost
      port: 5432
      credentials:
        username: admin
    cache:
      enabled: true
      ttl: 3600
features:
  - name: feature1
    enabled: true
  - name: feature2
    enabled: false
`

	overrideYAML := `
application:
  version: 2.0.0
  config:
    database:
      host: prod.db.example.com
      ssl: true
      credentials:
        username: prod_user
        password: secret
    monitoring:
      enabled: true
features:
  - name: feature1
    enabled: false
  - name: feature3
    enabled: true
`

	cfg := &config.Config{}
	svc := NewMergeService(cfg)
	
	merged, report, err := svc.MergeWithComments([]byte(baseYAML), []byte(overrideYAML))
	require.NoError(t, err)
	
	// Parse result
	var result map[string]interface{}
	err = yaml.Unmarshal(merged, &result)
	require.NoError(t, err)
	
	// Check merged structure
	app := result["application"].(map[string]interface{})
	assert.Equal(t, "myapp", app["name"]) // kept from base
	assert.Equal(t, "2.0.0", app["version"]) // updated from override
	
	appConfig := app["config"].(map[string]interface{})
	db := appConfig["database"].(map[string]interface{})
	assert.Equal(t, "prod.db.example.com", db["host"]) // updated
	assert.Equal(t, float64(5432), db["port"]) // kept from base
	assert.Equal(t, true, db["ssl"]) // added from override
	
	creds := db["credentials"].(map[string]interface{})
	assert.Equal(t, "prod_user", creds["username"]) // updated
	assert.Equal(t, "secret", creds["password"]) // added
	
	cache := appConfig["cache"].(map[string]interface{})
	assert.Equal(t, true, cache["enabled"]) // kept from base
	assert.Equal(t, float64(3600), cache["ttl"]) // kept from base
	
	monitoring := appConfig["monitoring"].(map[string]interface{})
	assert.Equal(t, true, monitoring["enabled"]) // added from override
	
	// Check report
	assert.Contains(t, report.AddedKeys, "application.config.database.ssl")
	assert.Contains(t, report.AddedKeys, "application.config.database.credentials.password")
	assert.Contains(t, report.AddedKeys, "application.config.monitoring")
	assert.Contains(t, report.UpdatedKeys, "application.version")
	assert.Contains(t, report.UpdatedKeys, "application.config.database.host")
	assert.Contains(t, report.UpdatedKeys, "features")
}