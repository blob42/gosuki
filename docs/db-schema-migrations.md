
To manage **database versions and schema migrations** in your Go application, you need a structured approach that ensures backward compatibility and allows for incremental schema changes. Below is a detailed solution tailored to your codebase:


### **1. Add a Version Tracking Mechanism**

Create a `schema_version` table to track the current version of the database. This table will store the version number, which you can increment as you introduce new migrations.

```go
const schemaVersionTable = `
CREATE TABLE IF NOT EXISTS schema_version (
    version INTEGER PRIMARY KEY
```


### **2. Initialize the Schema Version**

When initializing the database, check if the `schema_version` table exists. If not, create it and set the initial version.

```go
func (db *DB) InitSchema() error {
    // ... existing code ...

    // Create schema_version table if not exists
    _, err := db.Handle.Exec(schemaVersionTable)
    if err != nil {
        return DBError{DBName: db.Name, Err: err}
    }

    // Check current version
    var version int
    err = db.Handle.QueryRow("SELECT version FROM schema_version").Scan(&version)
    if err == sql.ErrNoRows {
        // First time: set version to 1
        _, err = db.Handle.Exec("INSERT INTO schema_version (version) VALUES (1)")
        if err != nil {
            return DBError{DBName: db.Name, Err: err}
        }
        version = 1
    }

    // Apply migrations if needed
    if version < 2 {
        if err := db.migrateToVersion2(); err != nil {
            return err
        }
        version = 2
    }

    if version < 3 {
        if err := db.migrateToVersion3(); err != nil {
            return err
        }
        version = 3
    }

    // Update the version in the schema_version table
    _, err = db.Handle.Exec("UPDATE schema_version SET version = ?", version)
    if err != nil {
        return DBError{DBName: db.Name, Err: err}
    }

    return nil
```


### **3. Define Migration Functions**

Each migration function should:
- Apply schema changes (e.g., `ALTER TABLE`, `ADD COLUMN`).
- Handle data migrations (e.g., populating new columns from existing data).
- Update the `schema_version` table.

#### Example: Migrate to Version 2 (Add a `tags` column)

```go
func (db *DB) migrateToVersion2() error {
    // Add a 'tags' column to the bookmarks table
    _, err := db.Handle.Exec("ALTER TABLE bookmarks ADD COLUMN tags TEXT DEFAULT ''")
    if err != nil {
        return DBError{DBName: db.Name, Err: err}
    }

    // Optional: Populate tags from existing data (if needed)
    // Example: Copy metadata to tags
    _, err = db.Handle.Exec("UPDATE bookmarks SET tags = metadata")
    if err != nil {
        return DBError{DBName: db.Name, Err: err}
    }

    return nil
```

#### Example: Migrate to Version 3 (Add a `description` column)

```go
func (db *DB) migrateToVersion3() error {
    // Add a 'description' column
    _, err := db.Handle.Exec("ALTER TABLE bookmarks ADD COLUMN description TEXT DEFAULT ''")
    if err != nil {
        return DBError{DBName: db.Name, Err: err}
    }

    return nil
```


### **4. Handle Migration Errors Gracefully**

Ensure that migrations are **idempotent** (safe to run multiple times) and include **rollback logic** if possible. For example:

```go
func (db *DB) migrateToVersion2() error {
    tx, err := db.Handle.Begin()
    if err != nil {
        return err
    }

    // Execute migration steps
    _, err = tx.Exec("ALTER TABLE bookmarks ADD COLUMN tags TEXT DEFAULT ''")
    if err != nil {
        tx.Rollback()
        return err
    }

    // Commit the transaction
    return tx.Commit()
```


### **5. Update `Init()` to Handle Migrations**

Modify the `Init()` method to call the migration logic:

```go
func (db *DB) Init() (*DB, error) {
    // ... existing code ...

    // Initialize schema and apply migrations
    if err := db.InitSchema(); err != nil {
        return nil, err
    }

    return db, nil
```


### **6. Example Migration Plan**

| Version | Description                     | SQL Changes                          |
|---------|----------------------------------|--------------------------------------|
| 1       | Initial schema                  | Create `bookmarks` table             |
| 2       | Add `tags` column               | `ALTER TABLE bookmarks ADD COLUMN tags TEXT DEFAULT ''` |
| 3       | Add `description` column        | `ALTER TABLE bookmarks ADD COLUMN description TEXT DEFAULT ''` |
| 4       | Rename `metadata` to `title`    | `ALTER TABLE bookmarks RENAME COLUMN metadata TO title` |


### **7. Best Practices**

- **Versioning**: Always increment the version number in `schema_version` after successful migrations.
- **Data Backups**: Before running migrations, back up the database (especially for production).
- **Testing**: Test migrations in a staging environment before deploying to production.
- **Documentation**: Document each migration step to explain the purpose and any data transformations.


### **8. Example Usage in `main.go`**

```go
func main() {
    db := NewDB("gosuki", GetDBPath(), DBTypeFileDSN)
    _, err := db.Init()
    if err != nil {
        log.Fatal(err)
    }
    // Proceed with database operations
```


### **Summary**


