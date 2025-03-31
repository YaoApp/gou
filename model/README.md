# Yao Model Module

A Go module for database model operations with TypeScript API support.

## Quick Example

Here's a simple example showing common model operations:

```typescript
// Find a record by ID
const user = Process("models.user.Find", 1, {});

// Get records with query parameters
const users = Process("models.user.Get", {
  wheres: [{ column: "status", value: "active" }],
  limit: 10,
});

// Create a new record
const newUser = Process("models.user.Create", {
  name: "John Doe",
  email: "john@example.com",
  status: "active",
});

// Update a record
Process("models.user.Update", 1, { status: "inactive" });

// Delete a record
Process("models.user.Delete", 1);
```

## Usage in TypeScript

You can use the Model module in TypeScript through the Process API. Below are examples of common operations with return type descriptions.

### Model Instance Operations

#### Find a record by ID

```typescript
/**
 * Finds a record by its ID
 * @param id - Record ID
 * @param params - Query parameters (optional)
 * @returns Record - The found record
 */
const user = Process("models.user.Find", 1, {
  select: ["id", "name", "email"],
  withs: { posts: {} },
});
```

#### Get records with query parameters

```typescript
/**
 * Gets records with the given query parameters
 * @param params - Query parameters
 * @returns Record[] - Array of records
 */
const users = Process("models.user.Get", {
  select: ["id", "name", "email"],
  wheres: [{ column: "status", value: "active" }],
  orders: [{ column: "created_at", direction: "desc" }],
  limit: 10,
});
```

#### Paginate records

```typescript
/**
 * Gets paginated records
 * @param params - Query parameters
 * @param page - Page number (1-based)
 * @param pageSize - Number of records per page
 * @returns object - Pagination result with data and pagination info
 */
const result = Process(
  "models.user.Paginate",
  {
    wheres: [{ column: "status", value: "active" }],
  },
  1,
  10
);
// Returns: { data: [...], pagination: { total: 100, page: 1, pagesize: 10, ... } }
```

#### Create a new record

```typescript
/**
 * Creates a new record
 * @param data - Record data
 * @returns Record - The created record with ID
 */
const newUser = Process("models.user.Create", {
  name: "John Doe",
  email: "john@example.com",
  status: "active",
});
```

#### Update a record

```typescript
/**
 * Updates a record by ID
 * @param id - Record ID
 * @param data - Data to update
 * @returns null
 */
Process("models.user.Update", 1, {
  name: "Updated Name",
  status: "inactive",
});
```

#### Save a record (Create or Update)

```typescript
/**
 * Saves a record (creates if no ID present, updates if ID exists)
 * @param data - Record data
 * @returns Record - The saved record with ID
 */
const user = Process("models.user.Save", {
  id: 1, // If ID exists, record will be updated
  name: "John Doe",
  email: "john@example.com",
});
```

#### Delete a record (Soft Delete)

```typescript
/**
 * Soft deletes a record by ID (sets deleted_at)
 * @param id - Record ID
 * @returns null
 */
Process("models.user.Delete", 1);
```

#### Destroy a record (Hard Delete)

```typescript
/**
 * Hard deletes a record by ID (removes from database)
 * @param id - Record ID
 * @returns null
 */
Process("models.user.Destroy", 1);
```

#### Insert multiple records

```typescript
/**
 * Inserts multiple records at once
 * @param columns - Array of column names
 * @param rows - Array of row data arrays
 * @returns null
 */
Process(
  "models.user.Insert",
  ["name", "email", "status"],
  [
    ["John Doe", "john@example.com", "active"],
    ["Jane Doe", "jane@example.com", "active"],
  ]
);
```

#### Update records with query

```typescript
/**
 * Updates records matching the query
 * @param params - Query parameters
 * @param data - Data to update
 * @returns number - Number of updated records
 */
const count = Process(
  "models.user.UpdateWhere",
  {
    wheres: [{ column: "status", value: "pending" }],
  },
  {
    status: "active",
  }
);
```

#### Delete records with query (Soft Delete)

```typescript
/**
 * Soft deletes records matching the query
 * @param params - Query parameters
 * @returns number - Number of deleted records
 */
const count = Process("models.user.DeleteWhere", {
  wheres: [{ column: "status", value: "inactive" }],
});
```

#### Destroy records with query (Hard Delete)

```typescript
/**
 * Hard deletes records matching the query
 * @param params - Query parameters
 * @returns number - Number of destroyed records
 */
const count = Process("models.user.DestroyWhere", {
  wheres: [{ column: "status", value: "inactive" }],
});
```

#### Save multiple records

```typescript
/**
 * Saves multiple records at once
 * @param rows - Array of records
 * @param commonData - Common data to apply to all records (optional)
 * @returns Record[] - The saved records with IDs
 */
const users = Process(
  "models.user.EachSave",
  [
    { name: "John", email: "john@example.com" },
    { name: "Jane", email: "jane@example.com" },
  ],
  {
    status: "active", // Will be applied to all records
  }
);
```

#### Delete and save records

```typescript
/**
 * Deletes records with specified IDs and then saves new records
 * @param ids - Array of IDs to delete
 * @param rows - Array of records to save
 * @param commonData - Common data to apply to all records (optional)
 * @returns Record[] - The saved records with IDs
 */
const users = Process(
  "models.user.EachSaveAfterDelete",
  [1, 2, 3], // IDs to delete
  [
    { name: "John", email: "john@example.com" },
    { name: "Jane", email: "jane@example.com" },
  ],
  { status: "active" } // Optional common data
);
```

#### Upsert a record

```typescript
/**
 * Inserts a record if it doesn't exist, updates it if it exists
 * @param data - Record data
 * @param uniqueBy - Column name(s) to check uniqueness
 * @param updateColumns - Columns to update if record exists (optional)
 * @returns any - ID of the upserted record
 */
const id = Process(
  "models.user.Upsert",
  { email: "john@example.com", name: "John Doe", status: "active" },
  "email", // Can be a string or array of strings
  ["name", "status"] // Optional columns to update
);
```

### Model Schema Operations

#### Migrate a model

```typescript
/**
 * Migrates the model schema to database
 * @param force - Whether to force recreation (optional, default: false)
 * @returns any - Migration result
 */
Process("models.user.Migrate", false);
```

#### Load a model

```typescript
/**
 * Loads a model from file or source
 * @param file - Model file path
 * @param source - Optional model source code (if provided)
 * @returns any - Load result
 */
// Load from file
Process("models.user.Load", "user.mod.yao");

// Load from source
const source = `{
  "name": "User",
  "table": { "name": "users", "comment": "Users table" },
  "columns": [
    { "label": "ID", "name": "id", "type": "ID" },
    { "label": "Name", "name": "name", "type": "string", "length": 80 }
  ]
}`;
Process("models.user.Load", "user.mod.yao", source);
```

#### Reload a model

```typescript
/**
 * Reloads a model from its file
 * @returns null
 */
Process("models.user.Reload");
```

#### Get model metadata

```typescript
/**
 * Gets the model metadata
 * @returns object - Model metadata
 */
const metadata = Process("models.user.Metadata");
```

#### Read model source

```typescript
/**
 * Reads the model source code
 * @returns string - Model source code
 */
const source = Process("models.user.Read");
```

#### Check if model exists

```typescript
/**
 * Checks if a model is loaded
 * @returns boolean - Whether the model exists
 */
const exists = Process("models.user.Exists");
```

### Snapshot Operations

#### Take a snapshot

```typescript
/**
 * Creates a snapshot of the model
 * @param inMemory - Whether to store in memory only (true) or in database (false)
 * @returns string - Snapshot name
 */
const snapshot = Process("models.user.TakeSnapshot", false);
```

#### Restore a snapshot

```typescript
/**
 * Restores a model from a snapshot
 * @param name - Snapshot name
 * @returns null
 */
Process("models.user.RestoreSnapshot", "user_snapshot_20230101");
```

#### Restore a snapshot by rename

```typescript
/**
 * Restores a model from a snapshot by renaming tables
 * @param name - Snapshot name
 * @returns null
 */
Process("models.user.RestoreSnapshotByRename", "user_snapshot_20230101");
```

#### Drop a snapshot

```typescript
/**
 * Drops a snapshot table
 * @param name - Snapshot name
 * @returns null
 */
Process("models.user.DropSnapshot", "user_snapshot_20230101");
```

#### Check if snapshot exists

```typescript
/**
 * Checks if a snapshot exists
 * @param name - Snapshot name
 * @returns boolean - Whether the snapshot exists
 */
const exists = Process("models.user.SnapshotExists", "user_snapshot_20230101");
```

### Global Model Operations

#### List all models

```typescript
/**
 * Lists all loaded models
 * @param options - List options
 * @returns object[] - Array of model information
 */
const models = Process("model.List", {
  metadata: true, // Include metadata (optional)
  columns: true, // Include columns (optional)
});
```

#### Get model DSL

```typescript
/**
 * Gets the model DSL (Domain Specific Language) definition
 * @param id - Model ID
 * @param options - DSL options
 * @returns object - Model DSL information
 */
const dsl = Process("model.DSL", "user", {
  metadata: true, // Include metadata (optional)
  columns: true, // Include columns (optional)
});
```

#### Check if model exists globally

```typescript
/**
 * Checks if a model exists in the global registry
 * @param id - Model ID
 * @returns boolean - Whether the model exists
 */
const exists = Process("model.Exists", "user");
```

#### Reload a model globally

```typescript
/**
 * Reloads a model in the global registry
 * @param id - Model ID
 * @returns null
 */
Process("model.Reload", "user");
```

#### Migrate a model globally

```typescript
/**
 * Migrates a model in the global registry
 * @param id - Model ID
 * @param force - Whether to force recreation (optional)
 * @returns any - Migration result
 */
Process("model.Migrate", "user", false);
```

#### Load a model globally

```typescript
/**
 * Loads a model in the global registry
 * @param id - Model ID
 * @param source - Model source code
 * @returns any - Load result
 */
Process("model.Load", "user", modelSourceJSON);
```

#### Unload a model globally

```typitten
/**
 * Unloads a model from the global registry
 * @param id - Model ID
 * @returns null
 */
Process("model.Unload", "user");
```

## Query Parameters Format

The query parameters object structure used in many operations:

```typescript
interface QueryParam {
  // Columns to select
  select?: string[] | any[];

  // Where conditions
  wheres?: QueryWhere[];

  // Order specifications
  orders?: QueryOrder[];

  // Related models to include
  withs?: { [key: string]: QueryParam };

  // Number of records to return
  limit?: number;

  // Number of records to skip
  offset?: number;

  // Group by columns
  groups?: string[];

  // Having conditions
  havings?: QueryWhere[];
}

interface QueryWhere {
  column: string;
  op?: string; // Default: "="
  value: any;
}

interface QueryOrder {
  column: string;
  direction?: string; // "asc" or "desc", default: "asc"
}
```

## Complete Workflow Example

```typescript
// Migration
Process("models.user.Migrate", false);

// Create a record
const user = Process("models.user.Create", {
  name: "John Doe",
  email: "john@example.com",
  status: "active",
});

// Find by ID
const foundUser = Process("models.user.Find", user.id, {});

// Update
Process("models.user.Update", user.id, { status: "vip" });

// Take a snapshot
const snapshot = Process("models.user.TakeSnapshot", false);
console.log("Snapshot created:", snapshot);

// Make more changes
Process("models.user.Update", user.id, { name: "John Smith" });

// Restore from snapshot
Process("models.user.RestoreSnapshot", snapshot);

// Query to verify restoration
const restoredUser = Process("models.user.Find", user.id, {});
console.log("Restored user:", restoredUser); // Should have original name

// Clean up snapshot
Process("models.user.DropSnapshot", snapshot);

// Delete the user
Process("models.user.Delete", user.id);
```

## Notes

- The model module supports both soft deletes (via `Delete` and `DeleteWhere`) and hard deletes (via `Destroy` and `DestroyWhere`).
- Use the appropriate snapshot operations carefully as they can affect database structure.
- When using `Upsert`, make sure to provide meaningful `uniqueBy` columns to determine record existence.
