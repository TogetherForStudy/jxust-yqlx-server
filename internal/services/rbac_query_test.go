package services

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"strings"
	"testing"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

type noopDriver struct{}

func (d noopDriver) Open(name string) (driver.Conn, error) {
	return noopConn{}, nil
}

type noopConnector struct{}

func (c noopConnector) Connect(ctx context.Context) (driver.Conn, error) {
	return noopConn{}, nil
}

func (c noopConnector) Driver() driver.Driver {
	return noopDriver{}
}

type noopConn struct{}

func (c noopConn) Prepare(query string) (driver.Stmt, error) {
	return noopStmt{}, nil
}

func (c noopConn) Close() error {
	return nil
}

func (c noopConn) Begin() (driver.Tx, error) {
	return noopTx{}, nil
}

type noopStmt struct{}

func (s noopStmt) Close() error {
	return nil
}

func (s noopStmt) NumInput() int {
	return -1
}

func (s noopStmt) Exec(args []driver.Value) (driver.Result, error) {
	return noopResult{}, nil
}

func (s noopStmt) Query(args []driver.Value) (driver.Rows, error) {
	return noopRows{}, nil
}

type noopTx struct{}

func (t noopTx) Commit() error {
	return nil
}

func (t noopTx) Rollback() error {
	return nil
}

type noopRows struct{}

func (r noopRows) Columns() []string {
	return nil
}

func (r noopRows) Close() error {
	return nil
}

func (r noopRows) Next(dest []driver.Value) error {
	return driver.ErrBadConn
}

type noopResult struct{}

func (r noopResult) LastInsertId() (int64, error) {
	return 0, nil
}

func (r noopResult) RowsAffected() (int64, error) {
	return 0, nil
}

func newDryRunDB(t *testing.T) *gorm.DB {
	t.Helper()

	sqlDB := sql.OpenDB(noopConnector{})
	t.Cleanup(func() {
		_ = sqlDB.Close()
	})

	db, err := gorm.Open(mysql.New(mysql.Config{
		Conn:                      sqlDB,
		SkipInitializeWithVersion: true,
	}), &gorm.Config{DryRun: true})
	if err != nil {
		t.Fatalf("open dry run db: %v", err)
	}

	return db
}

func TestRBACQueriesFilterSoftDeletedRecords(t *testing.T) {
	db := newDryRunDB(t)

	tests := []struct {
		name string
		sql  string
		want []string
	}{
		{
			name: "user role tags query",
			sql: db.ToSQL(func(tx *gorm.DB) *gorm.DB {
				var roleTags []string
				return userRoleTagsQuery(tx, 123).Find(&roleTags)
			}),
			want: []string{
				"roles.deleted_at is null",
				"ur.deleted_at is null",
			},
		},
		{
			name: "user permission tags query",
			sql: db.ToSQL(func(tx *gorm.DB) *gorm.DB {
				var permissionTags []string
				return userPermissionTagsQuery(tx, 123).Find(&permissionTags)
			}),
			want: []string{
				"permissions.deleted_at is null",
				"rp.deleted_at is null",
				"ur.deleted_at is null",
			},
		},
		{
			name: "users by role tags query",
			sql: db.ToSQL(func(tx *gorm.DB) *gorm.DB {
				return usersByRoleTagsQuery(tx, []string{"admin"}).Find(&[]struct{}{})
			}),
			want: []string{
				"users.deleted_at is null",
				"ur.deleted_at is null",
				"r.deleted_at is null",
			},
		},
		{
			name: "backoffice users by phone query",
			sql: db.ToSQL(func(tx *gorm.DB) *gorm.DB {
				var userIDs []uint
				return backofficeUsersByPhoneQuery(tx, "13800138000").Pluck("users.id", &userIDs)
			}),
			want: []string{
				"users.deleted_at is null",
				"ur.deleted_at is null",
				"roles.deleted_at is null",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sql := strings.ToLower(tt.sql)
			for _, want := range tt.want {
				if !strings.Contains(sql, want) {
					t.Fatalf("expected SQL to contain %q, got: %s", want, tt.sql)
				}
			}
		})
	}
}
