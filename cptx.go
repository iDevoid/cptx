package cptx

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"
)

type connectionString struct {
	main    string
	replica string
	domain  string
}

type ptxKey string
type ptx *sqlx.Tx

// Connections return the wrapped actions of postgres database
type Connections interface {
	Open() (Database, Transaction)
}

// Database contains both main and replica of database
type Database interface {
	// Main return only main db related functions
	Main() MainDB
	Replica() *sqlx.DB
}

// MainDB contains the function only for main db which can write data
type MainDB interface {
	// DB() *sqlx.DB

	// ExecuteMustTx is basically the ExecContext with only run using the transaction
	// this function is one option that allows your storage function query to be able to run only with declared transaction above the storage layer
	// means, if we found no transaction pointer inside the context, it won't run the query
	// the returns are basially same with ExecContext function, which are sql.Result and error
	ExecuteMustTx(ctx context.Context, query string, params map[string]interface{}) (sql.Result, error)

	// QueryRowMustTx is basically the QueryRowContext with only run using the transaction.
	// this function is one option that allows your storage function query to be able to run only with declared transaction above the storage layer.
	// means, if we found no transaction pointer inside the context, it won't run the query.
	// QueryRowMustTx allows you to scan the returning query as scans ...interface{} params,
	// and then return the error as the result
	QueryRowMustTx(ctx context.Context, query string, params map[string]interface{}, scans ...interface{}) error

	// Execute allows you to run the query without having a transaction declared above the storage level
	// meaning you can have a function that run with multiple purposes,
	// which are running with the transaction above the storage level
	// and running without the transaction for another domain
	// Execute wraps the ExecContext with sql.Result and error as the returns
	Execute(ctx context.Context, query string, params map[string]interface{}) (sql.Result, error)

	// QueryRow allows you to run the query without having a transaction declared above the storage level
	// meaning you can have a function that run with multiple purposes,
	// which are running with the transaction above the storage level
	// and running without the transaction for another domain
	// QueryRow also allows you to run the scan directly to the function on scans ...interface{} params
	// QueryRow wraps the QueryRowContext with error as the return
	QueryRow(ctx context.Context, query string, params map[string]interface{}, scans ...interface{}) error
}

type mainSQLX struct {
	db *sqlx.DB
}

// ExecuteMustTx is basically the ExecContext with only run using the transaction
// this function is one option that allows your storage function query to be able to run only with declared transaction above the storage layer
// means, if we found no transaction pointer inside the context, it won't run the query
// the returns are basially same with ExecContext function, which are sql.Result and error
func (m *mainSQLX) ExecuteMustTx(ctx context.Context, query string, params map[string]interface{}) (sql.Result, error) {
	iptx := ctx.Value(ptxKey("ptx"))
	exist := iptx != nil
	if !exist {
		return nil, fmt.Errorf("Transaction Pointer is not found inside the context")
	}

	query, args, err := sqlx.Named(query, params)
	if err != nil {
		return nil, err
	}
	query = m.db.Rebind(query)

	tx := iptx.(ptx)
	return tx.Tx.ExecContext(ctx, query, args...)
}

// Execute allows you to run the query without having a transaction declared above the storage level
// meaning you can have a function that run with multiple purposes,
// which are running with the transaction above the storage level
// and running without the transaction for another domain
// Execute wraps the ExecContext with sql.Result and error as the returns
func (m *mainSQLX) Execute(ctx context.Context, query string, params map[string]interface{}) (sql.Result, error) {
	iptx := ctx.Value(ptxKey("ptx"))
	exist := iptx != nil

	query, args, err := sqlx.Named(query, params)
	if err != nil {
		return nil, err
	}
	query = m.db.Rebind(query)

	if exist {
		tx := iptx.(ptx)
		return tx.Tx.ExecContext(ctx, query, args...)
	}
	return m.db.ExecContext(ctx, query, args...)
}

// QueryRowMustTx is basically the QueryRowContext with only run using the transaction.
// this function is one option that allows your storage function query to be able to run only with declared transaction above the storage layer.
// means, if we found no transaction pointer inside the context, it won't run the query.
// QueryRowMustTx allows you to scan the returning query as scans ...interface{} params,
// and then return the error as the result
func (m *mainSQLX) QueryRowMustTx(ctx context.Context, query string, params map[string]interface{}, scans ...interface{}) error {
	iptx := ctx.Value(ptxKey("ptx"))
	exist := iptx != nil
	if !exist {
		return fmt.Errorf("Transaction Pointer is not found inside the context")
	}

	query, args, err := sqlx.Named(query, params)
	if err != nil {
		return err
	}
	query = m.db.Rebind(query)

	tx := iptx.(ptx)
	return tx.Tx.QueryRowContext(ctx, query, args...).Scan(scans...)
}

// QueryRow allows you to run the query without having a transaction declared above the storage level
// meaning you can have a function that run with multiple purposes,
// which are running with the transaction above the storage level
// and running without the transaction for another domain
// QueryRow also allows you to run the scan directly to the function on scans ...interface{} params
// QueryRow wraps the QueryRowContext with error as the return
func (m *mainSQLX) QueryRow(ctx context.Context, query string, params map[string]interface{}, scans ...interface{}) error {
	iptx := ctx.Value(ptxKey("ptx"))
	exist := iptx != nil

	query, args, err := sqlx.Named(query, params)
	if err != nil {
		return err
	}
	query = m.db.Rebind(query)

	if exist {
		tx := iptx.(ptx)
		return tx.Tx.QueryRowContext(ctx, query, args...).Scan(scans...)
	}
	return m.db.QueryRowContext(ctx, query, args...).Scan(scans...)
}

// DB to get the sqlx database format
func (m *mainSQLX) DB() *sqlx.DB {
	return m.db
}

// databaseHolder where the opened database connection is being used
type databaseHolder struct {
	Mainx    *sqlx.DB
	Replicax *sqlx.DB
}

// Main return only main db related functions
func (d *databaseHolder) Main() MainDB {
	return &mainSQLX{
		db: d.Mainx,
	}
}

// Replica retuns the database itself
func (d *databaseHolder) Replica() *sqlx.DB {
	return d.Replicax
}

// Transaction is for the import to usecase or repository layer
type Transaction interface {
	Begin(ctx *context.Context) (Tx, error)
}

// Tx contains the transaction related functions for being used inside repository/data-logic or usecase layer that combines multi or across domain storage functions
type Tx interface {
	Commit() error
	Rollback() error
}

// uniqueHolder is only about the specific transaction being made on Begin
// those multiple tx should be stored as unique, individual transaction
// so you won't get different transaction when doing the operation on rollback and commit
type uniqueHolder struct {
	// key string
	tx *sqlx.Tx
}

// Begin opens the transaction and save to tx collection returning.
// context that has the key for accessing the transaction.
// Tx contains the transaction operation such as commit and rollback.
// error that will be returned only if the transaction failed to begin.
// the pointer of transaction is being included inside the context
// the reason being is we usually bring context to whatever the function is
func (d *databaseHolder) Begin(ctx *context.Context) (Tx, error) {
	tx, err := d.Mainx.Beginx()
	if err != nil {
		return nil, err
	}
	newCtx := context.WithValue(*ctx, ptxKey("ptx"), ptx(tx))
	*ctx = newCtx
	return &uniqueHolder{
		tx,
	}, nil
}

// Commit wraps the commit of transaction and executing it based on key registered on tx connection
func (uh *uniqueHolder) Commit() error {
	return uh.tx.Commit()
}

// Commit wraps the rollback of transaction and executing it based on key registered on tx connection
func (uh *uniqueHolder) Rollback() error {
	return uh.tx.Rollback()
}

// Initialize is to initTx the postgres platform with connection string both main or replica.
// use the main connection string if there's no replica database, both must be the same connection string.
// never set it to empty string, it will cause the fatal and stops the entire app where the database is being initialize.
func Initialize(main, replica, domain string) Connections {
	return &connectionString{
		main:    main,
		replica: replica,
		domain:  domain,
	}
}

// why is this needed? because sonarqube will say there's a duplication if you write it multiple times
// you know, a good writing habit tho
var stringType = "type"
var stringConnection = "connection"
var postgres = "postgres"

// Open is creating the database postgres connections (main and replica) and special transaction level for repository layer
func (cs *connectionString) Open() (Database, Transaction) {
	// log fieds for logrus, no need to write this multiple times
	logFields := logrus.Fields{
		"platform": postgres,
		"domain":   cs.domain,
	}
	logMainFields := logrus.Fields{
		stringType:       "main",
		stringConnection: cs.main,
	}
	logReplicaFields := logrus.Fields{
		stringType:       "replica",
		stringConnection: cs.replica,
	}
	logrus.WithFields(logFields).Info("Connecting to PostgreSQL DB")
	logrus.WithFields(logFields).Info("Opening Connection to Main")
	dbMain, err := sqlx.Open(postgres, cs.main)
	if err != nil {
		logrus.WithFields(logMainFields).Fatal(err)
		panic(err)
	}
	err = dbMain.Ping()
	if err != nil {
		logrus.WithFields(logMainFields).Fatal(err)
		panic(err)
	}
	logrus.WithFields(logFields).Info("Opening Connection to Replica")
	dbReplica, err := sqlx.Open(postgres, cs.main)
	if err != nil {
		logrus.WithFields(logReplicaFields).Fatal(err)
		panic(err)
	}
	err = dbReplica.Ping()
	if err != nil {
		logrus.WithFields(logReplicaFields).Fatal(err)
		panic(err)
	}
	return &databaseHolder{
			Mainx:    dbMain,
			Replicax: dbReplica,
		},
		&databaseHolder{
			Mainx: dbMain,
		}
}
