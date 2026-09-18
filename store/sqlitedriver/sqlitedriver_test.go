package sqlitedriver

import (
	"database/sql"
	"path/filepath"
	"testing"

	"gorm.io/gorm"
)

// TestSingleRegistration pins the invariant this package exists for: exactly
// one "sqlite" driver is registered in the test binary. (A second registration
// would panic at init, before any test ran — this test documents the
// expectation and guards against a future driver being registered under a
// different name so that Open silently stops matching.)
func TestSingleRegistration(t *testing.T) {
	n := 0
	for _, d := range sql.Drivers() {
		if d == DriverName {
			n++
		}
	}
	if n != 1 {
		t.Fatalf("driver %q registered %d times, want 1 (drivers=%v backend=%s)", DriverName, n, sql.Drivers(), Backend())
	}
}

func TestOpenReadWrite(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(`CREATE TABLE kv (k TEXT PRIMARY KEY, v TEXT)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO kv (k, v) VALUES (?, ?)`, "backend", Backend()); err != nil {
		t.Fatal(err)
	}
	var v string
	if err := db.QueryRow(`SELECT v FROM kv WHERE k = ?`, "backend").Scan(&v); err != nil {
		t.Fatal(err)
	}
	if v != Backend() {
		t.Fatalf("read %q, want %q", v, Backend())
	}
}

type row struct {
	ID   uint `gorm:"primaryKey"`
	Name string
}

func TestGormDialectorReadWrite(t *testing.T) {
	db, err := gorm.Open(GormDialector(filepath.Join(t.TempDir(), "g.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&row{}); err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&row{Name: "one"}).Error; err != nil {
		t.Fatal(err)
	}
	var got row
	if err := db.First(&got).Error; err != nil {
		t.Fatal(err)
	}
	if got.Name != "one" {
		t.Fatalf("got %+v", got)
	}
}
