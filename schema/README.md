# Yao Schema Module

A Go module for database schema operations with TypeScript API support.

## Quick Example

Here's a simple example showing common schema operations:

```typescript
// Create a schema
Process("schemas.default.Create", "my_database");

// Create a table
Process("schemas.default.TableCreate", "users", {
  name: "users",
  columns: [
    { name: "id", type: "ID" },
    { name: "name", type: "string", length: 80 },
    { name: "email", type: "string", index: true },
  ],
});

// Add a new column
Process("schemas.default.ColumnAdd", "users", {
  name: "status",
  type: "string",
  length: 20,
});

// Add an index
Process("schemas.default.IndexAdd", "users", {
  name: "status_index",
  type: "index",
  columns: ["status"],
});
```

## Table Blueprint Specification

When creating or modifying tables, you need to provide a blueprint that defines the table structure. Here's the complete specification based on the source code:

```typescript
interface Blueprint {
  columns: Column[]; // Array of column definitions
  indexes?: Index[]; // Optional array of index definitions
  option?: BlueprintOption; // Optional table options
  temporary?: boolean; // If true, the table will be created as a temporary table (in memory)
}

interface BlueprintOption {
  timestamps?: boolean; // Add created_at, updated_at fields
  soft_deletes?: boolean; // Add deleted_at field
  trackings?: boolean; // Add created_by, updated_by, deleted_by fields
  constraints?: boolean; // Add constraints definition
  permission?: boolean; // Add __permission fields
  logging?: boolean; // Add __logging_id fields
  read_only?: boolean; // Ignore the migrate operation
}

interface Column {
  name: string; // Column name
  label?: string; // Column label
  type?: string; // Column type
  title?: string; // Column title
  description?: string; // Column description
  comment?: string; // Column comment
  length?: number; // Column length
  precision?: number; // Numeric precision
  scale?: number; // Numeric scale
  nullable?: boolean; // Whether the column can be null
  option?: string[]; // Column options
  default?: any; // Default value
  default_raw?: string; // Raw default value
  generate?: string; // Value generation method (Increment, UUID, ...)
  crypt?: string; // Encryption method (AES, PASSWORD, AES-256, AES-128, PASSWORD-HASH, ...)
  index?: boolean; // Whether to create an index
  unique?: boolean; // Whether the column value should be unique
  primary?: boolean; // Whether this is a primary key
  origin?: string; // Original column name
}

interface Index {
  comment?: string; // Index comment
  name?: string; // Index name
  columns: string[]; // Array of column names to include in the index
  type?: string; // Index type (primary, unique, index, match)
  origin?: string; // Original index name
}
```

Example of a complete table blueprint:

```typescript
const userTableBlueprint: Blueprint = {
  columns: [
    {
      name: "id",
      type: "ID",
      primary: true,
      comment: "Primary key",
    },
    {
      name: "username",
      type: "string",
      length: 80,
      nullable: false,
      unique: true,
      comment: "User login name",
    },
    {
      name: "email",
      type: "string",
      length: 255,
      nullable: false,
      index: true,
      comment: "User email address",
    },
    {
      name: "password",
      type: "string",
      length: 255,
      crypt: "PASSWORD-HASH",
      comment: "User password",
    },
  ],
  indexes: [
    {
      name: "email_username_idx",
      type: "index",
      columns: ["email", "username"],
      comment: "Composite index for email and username",
    },
  ],
  option: {
    timestamps: true, // Adds created_at and updated_at
    soft_deletes: true, // Adds deleted_at
    trackings: true, // Adds created_by, updated_by, deleted_by
  },
  temporary: false,
};

Process("schemas.default.TableCreate", "users", userTableBlueprint);
```

Common Column Types:

- `ID`: Auto-incrementing primary key
- `string`: Variable-length string
- `integer`: Integer number
- `float`: Floating-point number
- `decimal`: Decimal number
- `datetime`: Date and time
- `date`: Date only
- `time`: Time only
- `boolean`: Boolean value
- `json`: JSON data
- `text`: Long text
- `longtext`: Very long text
- `binary`: Binary data

## Usage in TypeScript

You can use the Schema module in TypeScript through the Process API. Below are examples of common operations with parameter descriptions.

### Schema Operations

#### Create a Schema

```typescript
/**
 * Creates a new database schema
 * @param name - Schema name
 * @returns null
 */
Process("schemas.default.Create", "my_database");
```

#### Drop a Schema

```typescript
/**
 * Drops a database schema
 * @param name - Schema name
 * @returns null
 */
Process("schemas.default.Drop", "my_database");
```

### Table Operations

#### List Tables

```typescript
/**
 * Gets the list of tables
 * @param prefix - Optional table name prefix to filter results
 * @returns string[] - Array of table names
 */
const tables = Process("schemas.default.Tables", "prefix_");
// or without prefix
const allTables = Process("schemas.default.Tables");
```

#### Check Table Existence

```typescript
/**
 * Checks if a table exists
 * @param tableName - Name of the table
 * @returns boolean - Whether the table exists
 */
const exists = Process("schemas.default.TableExists", "users");
```

#### Get Table Blueprint

```typescript
/**
 * Gets a table's blueprint/schema definition
 * @param tableName - Name of the table
 * @returns object - Table blueprint
 */
const blueprint = Process("schemas.default.TableGet", "users");
```

#### Create Table

```typescript
/**
 * Creates a new table
 * @param tableName - Name of the table
 * @param blueprint - Table blueprint definition
 * @returns null
 */
Process("schemas.default.TableCreate", "users", {
  name: "users",
  columns: [
    { name: "id", type: "ID" },
    { name: "name", type: "string", length: 80 },
  ],
});
```

#### Drop Table

```typescript
/**
 * Drops a table
 * @param tableName - Name of the table
 * @returns null
 */
Process("schemas.default.TableDrop", "users");
```

#### Rename Table

```typescript
/**
 * Renames a table
 * @param tableName - Current table name
 * @param newTableName - New table name
 * @returns null
 */
Process("schemas.default.TableRename", "users", "users_new");
```

#### Compare Tables

```typescript
/**
 * Compares two table blueprints and returns the differences
 * @param blueprint1 - First table blueprint
 * @param blueprint2 - Second table blueprint
 * @returns object - Differences between the tables
 */
const diff = Process("schemas.default.TableDiff", blueprint1, blueprint2);
```

#### Save Table

```typescript
/**
 * Saves a table (creates if not exists, updates if exists)
 * @param tableName - Name of the table
 * @param blueprint - Table blueprint definition
 * @returns null
 */
Process("schemas.default.TableSave", "users", {
  name: "users",
  columns: [
    { name: "id", type: "ID" },
    { name: "name", type: "string", length: 80 },
  ],
});
```

### Column Operations

#### Add Column

```typescript
/**
 * Adds a new column to a table
 * @param tableName - Name of the table
 * @param column - Column definition
 * @returns null
 */
Process("schemas.default.ColumnAdd", "users", {
  name: "email",
  type: "string",
  length: 255,
  index: true,
});
```

#### Alter Column

```typescript
/**
 * Alters an existing column (adds if not exists)
 * @param tableName - Name of the table
 * @param column - Column definition
 * @returns null
 */
Process("schemas.default.ColumnAlt", "users", {
  name: "email",
  type: "string",
  length: 320,
  index: true,
});
```

#### Delete Column

```typescript
/**
 * Deletes a column from a table
 * @param tableName - Name of the table
 * @param columnName - Name of the column
 * @returns null
 */
Process("schemas.default.ColumnDel", "users", "email");
```

### Index Operations

#### Add Index

```typescript
/**
 * Adds a new index to a table
 * @param tableName - Name of the table
 * @param index - Index definition
 * @returns null
 */
Process("schemas.default.IndexAdd", "users", {
  name: "email_status_index",
  type: "index",
  columns: ["email", "status"],
});
```

#### Delete Index

```typescript
/**
 * Deletes an index from a table
 * @param tableName - Name of the table
 * @param indexName - Name of the index
 * @returns null
 */
Process("schemas.default.IndexDel", "users", "email_status_index");
```

## Complete Workflow Example

```typescript
// Create users table
Process("schemas.default.TableCreate", "users", {
  name: "users",
  columns: [
    { name: "id", type: "ID" },
    { name: "name", type: "string", length: 80 },
    { name: "email", type: "string", length: 255 },
  ],
});

// Add a status column
Process("schemas.default.ColumnAdd", "users", {
  name: "status",
  type: "string",
  length: 20,
});

// Add a composite index
Process("schemas.default.IndexAdd", "users", {
  name: "email_status_idx",
  type: "index",
  columns: ["email", "status"],
});

// Verify table exists
const exists = Process("schemas.default.TableExists", "users");
console.log("Table exists:", exists);

// Get table blueprint
const blueprint = Process("schemas.default.TableGet", "users");
console.log("Table blueprint:", blueprint);

// Rename table
Process("schemas.default.TableRename", "users", "users_new");

// Drop table
Process("schemas.default.TableDrop", "users_new");

// Drop schema
Process("schemas.default.Drop", "my_app");
```

## Notes

- All operations are executed through the Process API using the `schemas.<connector>` namespace
- The default connector is named "default", but you can use other configured database connectors
- Table blueprints should follow the schema specification format
- Index operations support both single-column and composite indexes
- Schema operations may be restricted based on database user permissions
