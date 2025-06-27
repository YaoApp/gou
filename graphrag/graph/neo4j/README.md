# Neo4j GraphStore

Neo4j GraphStore implements the GraphRAG graph database interface, supporting two storage modes:

## Storage Modes

### 1. Separate Database Mode (Enterprise Edition)

Each graph uses a separate Neo4j database, providing complete isolation.

### 2. Label-based Mode (Community & Enterprise Edition)

Uses labels and namespaces within a single database to distinguish different graphs.

## Configuration Options

### Basic Configuration

```go
config := types.GraphStoreConfig{
    StoreType:   "neo4j",
    DatabaseURL: "neo4j://localhost:7687",
    DriverConfig: map[string]interface{}{
        "username": "neo4j",
        "password": "password",
        "use_separate_database": false, // Use label-based mode
    },
}
```

### Custom Prefix Configuration

In label-based mode, you can customize the prefixes used by the system to avoid conflicts with business data:

```go
config := types.GraphStoreConfig{
    StoreType:   "neo4j",
    DatabaseURL: "neo4j://localhost:7687",
    DriverConfig: map[string]interface{}{
        "username": "neo4j",
        "password": "password",

        // Storage mode configuration
        "use_separate_database": false,

        // Custom prefix configuration (optional)
        "graph_label_prefix":       "MyApp_",           // Default: "__Graph_"
        "graph_namespace_property": "__my_namespace",   // Default: "__graph_namespace"
    },
}
```

### Configuration Reference

| Configuration Key          | Type   | Default Value         | Description                                                         |
| -------------------------- | ------ | --------------------- | ------------------------------------------------------------------- |
| `use_separate_database`    | bool   | false                 | Whether to use separate database mode (requires Enterprise Edition) |
| `graph_label_prefix`       | string | `"__Graph_"`          | Graph label prefix in label-based mode                              |
| `graph_namespace_property` | string | `"__graph_namespace"` | Namespace property name in label-based mode                         |

### Prefix Usage Examples

With default prefixes:

- Graph name: `myapp`
- Generated label: `__Graph_myapp`
- Namespace property: `__graph_namespace`

With custom prefixes:

- Graph name: `myapp`
- Custom prefix: `MyCompany_`
- Generated label: `MyCompany_myapp`
- Custom property: `__my_namespace`

## Usage Example

```go
package main

import (
    "context"
    "fmt"
    "github.com/yaoapp/gou/graphrag/graph/neo4j"
    "github.com/yaoapp/gou/graphrag/types"
)

func main() {
    store := neo4j.NewStore()

    config := types.GraphStoreConfig{
        StoreType:   "neo4j",
        DatabaseURL: "neo4j://localhost:7687",
        DriverConfig: map[string]interface{}{
            "username": "neo4j",
            "password": "password",

            // Use custom prefixes to avoid conflicts
            "graph_label_prefix":       "MyApp_",
            "graph_namespace_property": "__myapp_ns",
        },
    }

    ctx := context.Background()

    // Connect
    err := store.Connect(ctx, config)
    if err != nil {
        panic(err)
    }
    defer store.Close()

    // Create graph
    err = store.CreateGraph(ctx, "knowledge", nil)
    if err != nil {
        panic(err)
    }

    // Neo4j will now use label "MyApp_knowledge"
    // and property "__myapp_ns" to organize data

    fmt.Println("Graph created successfully!")
}
```

## Best Practices

1. **Production Environment**: Recommend using custom prefixes to avoid conflicts with business data
2. **Prefix Naming**: Use double underscore prefixes (e.g., `__MyApp_`) to clearly identify system data
3. **Consistency**: Maintain consistent prefix configuration across the same application
4. **Documentation**: Document custom prefix conventions within your team

## Error Handling

The system automatically detects Neo4j version and validates configuration:

- If `use_separate_database: true` is set on Community Edition, an error will be returned
- If prefix configuration is empty string, default values will be used automatically
- Runtime validation ensures configuration compatibility
