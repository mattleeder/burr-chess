package models

import (
	"database/sql"
	"errors"
	"time"
)

const (
	queryRetryDelay = 50 * time.Millisecond
	maxQueryRetries = 5
)

func ExecStatementWithRetry(stmt *sql.Stmt, args ...any) (sql.Result, error) {

	var result sql.Result
	var err error

	for range maxQueryRetries {
		result, err = stmt.Exec(args...)
		if err == nil {
			return result, nil
		} else if err.Error() == "database is locked (5) (SQLITE_BUSY)" {
			app.errorLog.Printf("%v, sleeping for %s\n", err.Error(), queryRetryDelay)
			time.Sleep(queryRetryDelay)
			continue
		} else {
			return nil, err
		}
	}

	return nil, errors.New("ExecStatementWithRetry: max retries exceeded")
}

func QueryWithRetry(DB *sql.DB, query string, args ...any) (*sql.Rows, error) {

	var rows *sql.Rows
	var err error

	for range maxQueryRetries {
		rows, err = DB.Query(query, args...)
		if err == nil {
			return rows, nil
		} else if err.Error() == "database is locked (5) (SQLITE_BUSY)" {
			app.errorLog.Printf("%v, sleeping for %s\n", err.Error(), queryRetryDelay)
			time.Sleep(queryRetryDelay)
			continue
		} else {
			return nil, err
		}
	}

	return nil, errors.New("QueryWithRetry: max retries exceeded")
}

func QueryRowWithRetry(DB *sql.DB, query string, queryArgs []any, scanArgs []any) error {

	var err error

	for range maxQueryRetries {
		row := DB.QueryRow(query, queryArgs...)
		err = row.Scan(scanArgs...)
		if err == nil {
			return nil
		} else if errors.Is(err, sql.ErrNoRows) {
			return err
		} else if err.Error() == "database is locked (5) (SQLITE_BUSY)" {
			app.errorLog.Printf("%v, sleeping for %s\n", err.Error(), queryRetryDelay)
			time.Sleep(queryRetryDelay)
			continue
		} else {
			return err
		}
	}

	return errors.New("ScanRowWithRetry: max retries exceeded")
}
