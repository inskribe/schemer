/*
Copyright © 2025 Roy Sowers <inskribe@inskribestudio.com>

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/

package utils

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"

	"github.com/jackc/pgx/v5"
	_ "github.com/jackc/pgx/v5/stdlib"

	er "github.com/inskribe/schemer/internal/errschemer"
	"github.com/inskribe/schemer/internal/glog"
)

// ConnectDatabase opens and verifies a PostgreSQL connection using the given connection string.
// Uses the pgx driver and ensures the connection is valid by performing a ping.
//
// Params:
//   - connString: PostgreSQL connection string
//
// Returns:
//   - *sql.DB: an open and verified database connection
//   - error: non-nil if connection or ping fails
func ConnectDatabase(connString string) (*sql.DB, error) {
	DB, err := sql.Open("pgx", connString)
	if err != nil {
		return nil, &er.SchemerErr{
			Code:    "0009",
			Message: "failed to establishe a connection with database.",
			Err:     err,
		}
	}

	if err = DB.Ping(); err != nil {
		return nil, &er.SchemerErr{
			Code:    "0010",
			Message: "failed to ping database.",
			Err:     err,
		}

	}
	return DB, nil
}

// WithConn establishes a pgx connection and executes the provided function with it.
// Automatically handles connection opening, context setup, and deferred closing.
//
// Params:
//   - connString: PostgreSQL connection string
//   - fn: a callback function that receives the opened connection and context
//
// Returns:
//   - error: any error encountered during connection or from the callback execution
var WithConn = func(connString string, fn func(*pgx.Conn, context.Context) error) error {
	ctx := context.Background()
	connection, err := pgx.Connect(ctx, connString)
	if err != nil {
		return &er.SchemerErr{
			Code:    "0008",
			Message: "failed to conntect with database.",
			Err:     err,
		}
	}
	defer connection.Close(ctx)

	return fn(connection, ctx)
}

// CreateSchemerTable creates the schemer tracking table if it does not already exist.
// Reads the table schema from schemer.sql located in the deltas directory.
//
// Params:
//   - database: pointer to an open pgx.Conn
//   - ctx: context for executing the database operations
//
// Returns:
//   - error: non-nil if the table check, file read, or table creation fails
func CreateSchemerTable(database *pgx.Conn, ctx context.Context) error {
	if database == nil {
		return &er.SchemerErr{
			Code:    "0004",
			Message: "recived nil database pointer."}
	}

	deltasPath, err := GetDeltaPath()
	if err != nil {
		return err
	}

	statment, err := os.ReadFile(filepath.Join(deltasPath, "schemer.sql"))
	if err != nil {
		return &er.SchemerErr{
			Code:    "0005",
			Message: "failed to read schemer.sql from deltas directory.",
			Err:     err,
		}
	}

	var exists bool
	err = database.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT FROM information_schema.tables 
			WHERE table_schema = 'public' 
			AND table_name = 'schemer'
		);
	`).Scan(&exists)
	if err != nil {
		return &er.SchemerErr{
			Code:    "0006",
			Message: "failed to query for table during pre-creation check.",
			Err:     err,
		}
	}

	if exists {
		glog.Info("Schemer table already exists. Skipping table creation")
		return nil
	}

	_, err = database.Exec(ctx, string(statment))
	if err != nil {
		return &er.SchemerErr{
			Code:    "0007",
			Message: "failed to create schemer table.",
			Err:     err,
		}

	}

	glog.Info("Schemer table created successfuly.")

	return nil
}
