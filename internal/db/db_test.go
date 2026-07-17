package db

import (
	"testing"
)

func TestSchemaUsesSingleEncryptedStorageModel(t *testing.T) {
	database, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	if err := database.InitSchema(); err != nil {
		t.Fatal(err)
	}
	if err := database.InitSchema(); err != nil {
		t.Fatalf("schema is not idempotent: %v", err)
	}
	version, err := database.SchemaVersion()
	if err != nil {
		t.Fatal(err)
	}
	if version != CurrentSchemaVersion {
		t.Fatalf("schema version = %d, want %d", version, CurrentSchemaVersion)
	}

	requiredTables := []string{
		"local_identity",
		"local_messages",
		"relay_cache",
		"contribution_receipts",
		"credit_states",
		"witness_markers",
	}
	for _, table := range requiredTables {
		var count int
		if err := database.SQL().QueryRow(
			"SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = ?",
			table,
		).Scan(&count); err != nil {
			t.Fatal(err)
		}
		if count != 1 {
			t.Fatalf("required table %q missing", table)
		}
	}

	rows, err := database.SQL().Query("PRAGMA table_info(local_messages)")
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		var cid, notNull, primaryKey int
		var name, typeName string
		var defaultValue any
		if err := rows.Scan(&cid, &name, &typeName, &notNull, &defaultValue, &primaryKey); err != nil {
			t.Fatal(err)
		}
		if name == "plaintext" {
			t.Fatal("local_messages must not contain a plaintext column")
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
}
