package stdlib

import (
	"database/sql"
	"fmt"
	"math"
	"strings"

	"github.com/issueye/goscript/internal/ast"
	"github.com/issueye/goscript/internal/module"
	"github.com/issueye/goscript/internal/object"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	_ "github.com/microsoft/go-mssqldb"
	_ "modernc.org/sqlite"
)

type dbConn struct {
	driver string
	dsn    string
	db     *sql.DB
}

type dbTx struct {
	tx *sql.Tx
}

type dbStmt struct {
	stmt *sql.Stmt
}

type dbExecutor interface {
	Exec(query string, args ...interface{}) (sql.Result, error)
	Query(query string, args ...interface{}) (*sql.Rows, error)
}

func init() {
	module.RegisterNative("@std/db", func(env *object.Environment) (object.Object, error) {
		exports := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
		initDBModule(exports)
		return exports, nil
	})
}

func initDBModule(exports *object.Hash) {
	setHashMember(exports, "open", &object.Builtin{Name: "db.open", Fn: dbOpen})
	setHashMember(exports, "drivers", strSliceToArray([]string{"sqlite", "sqlite3", "postgres", "pg", "mysql", "mssql", "sqlserver"}))
}

func dbOpen(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	driver, errObj := requiredString(pos, "db.open", args, 0, "driver")
	if errObj != nil {
		return errObj
	}
	dsn, errObj := requiredString(pos, "db.open", args, 1, "dsn")
	if errObj != nil {
		return errObj
	}
	driverName, err := normalizeDBDriver(driver)
	if err != nil {
		return object.NewError(pos, "db.open: %v", err)
	}
	db, err := sql.Open(driverName, dsn)
	if err != nil {
		return object.NewError(pos, "db.open: %v", err)
	}
	conn := &dbConn{driver: driverName, dsn: dsn, db: db}
	return dbConnectionObject(conn)
}

func dbConnectionObject(conn *dbConn) *object.Hash {
	obj := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
	setHashMember(obj, "__db", &object.GoObject{Value: conn})
	setHashMember(obj, "driver", &object.String{Value: conn.driver})
	setHashMember(obj, "dsn", &object.String{Value: conn.dsn})
	setHashMember(obj, "exec", &object.Builtin{Name: "db.exec", Fn: dbExec, Extra: &object.GoObject{Value: conn}})
	setHashMember(obj, "query", &object.Builtin{Name: "db.query", Fn: dbQuery, Extra: &object.GoObject{Value: conn}})
	setHashMember(obj, "queryOne", &object.Builtin{Name: "db.queryOne", Fn: dbQueryOne, Extra: &object.GoObject{Value: conn}})
	setHashMember(obj, "prepare", &object.Builtin{Name: "db.prepare", Fn: dbPrepare, Extra: &object.GoObject{Value: conn}})
	setHashMember(obj, "begin", &object.Builtin{Name: "db.begin", Fn: dbBegin, Extra: &object.GoObject{Value: conn}})
	setHashMember(obj, "setMaxOpenConns", &object.Builtin{Name: "db.setMaxOpenConns", Fn: dbSetMaxOpenConns, Extra: &object.GoObject{Value: conn}})
	setHashMember(obj, "setMaxIdleConns", &object.Builtin{Name: "db.setMaxIdleConns", Fn: dbSetMaxIdleConns, Extra: &object.GoObject{Value: conn}})
	setHashMember(obj, "ping", &object.Builtin{Name: "db.ping", Fn: dbPing, Extra: &object.GoObject{Value: conn}})
	setHashMember(obj, "close", &object.Builtin{Name: "db.close", Fn: dbClose, Extra: &object.GoObject{Value: conn}})
	return obj
}

func dbExec(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	conn, errObj := boundDB(pos, env, "db.exec")
	if errObj != nil {
		return errObj
	}
	query, params, errObj := dbQueryArgs(pos, "db.exec", args)
	if errObj != nil {
		return errObj
	}
	result, err := conn.db.Exec(query, params...)
	if err != nil {
		return object.NewError(pos, "db.exec: %v", err)
	}
	return sqlResultObject(result)
}

func dbQuery(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	conn, errObj := boundDB(pos, env, "db.query")
	if errObj != nil {
		return errObj
	}
	query, params, errObj := dbQueryArgs(pos, "db.query", args)
	if errObj != nil {
		return errObj
	}
	rows, err := conn.db.Query(query, params...)
	if err != nil {
		return object.NewError(pos, "db.query: %v", err)
	}
	defer rows.Close()
	result, err := rowsToArray(rows)
	if err != nil {
		return object.NewError(pos, "db.query: %v", err)
	}
	return result
}

func dbQueryOne(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	result := dbQuery(env, pos, args...)
	if object.IsRuntimeError(result) {
		return result
	}
	rows, ok := result.(*object.Array)
	if !ok || len(rows.Elements) == 0 {
		return object.NULL
	}
	return rows.Elements[0]
}

func dbPrepare(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	conn, errObj := boundDB(pos, env, "db.prepare")
	if errObj != nil {
		return errObj
	}
	query, errObj := requiredString(pos, "db.prepare", args, 0, "query")
	if errObj != nil {
		return errObj
	}
	stmt, err := conn.db.Prepare(query)
	if err != nil {
		return object.NewError(pos, "db.prepare: %v", err)
	}
	return dbStmtObject(&dbStmt{stmt: stmt})
}

func dbBegin(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	conn, errObj := boundDB(pos, env, "db.begin")
	if errObj != nil {
		return errObj
	}
	tx, err := conn.db.Begin()
	if err != nil {
		return object.NewError(pos, "db.begin: %v", err)
	}
	return dbTxObject(&dbTx{tx: tx})
}

func dbSetMaxOpenConns(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	conn, errObj := boundDB(pos, env, "db.setMaxOpenConns")
	if errObj != nil {
		return errObj
	}
	n, errObj := requiredInt(pos, "db.setMaxOpenConns", args, 0, "count")
	if errObj != nil {
		return errObj
	}
	conn.db.SetMaxOpenConns(n)
	return object.UNDEFINED
}

func dbSetMaxIdleConns(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	conn, errObj := boundDB(pos, env, "db.setMaxIdleConns")
	if errObj != nil {
		return errObj
	}
	n, errObj := requiredInt(pos, "db.setMaxIdleConns", args, 0, "count")
	if errObj != nil {
		return errObj
	}
	conn.db.SetMaxIdleConns(n)
	return object.UNDEFINED
}

func dbPing(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	conn, errObj := boundDB(pos, env, "db.ping")
	if errObj != nil {
		return errObj
	}
	if err := conn.db.Ping(); err != nil {
		return object.NewError(pos, "db.ping: %v", err)
	}
	return object.TRUE
}

func dbClose(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	conn, errObj := boundDB(pos, env, "db.close")
	if errObj != nil {
		return errObj
	}
	if err := conn.db.Close(); err != nil {
		return object.NewError(pos, "db.close: %v", err)
	}
	return object.UNDEFINED
}

func dbTxObject(tx *dbTx) *object.Hash {
	obj := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
	setHashMember(obj, "__tx", &object.GoObject{Value: tx})
	setHashMember(obj, "exec", &object.Builtin{Name: "db.tx.exec", Fn: dbTxExec, Extra: &object.GoObject{Value: tx}})
	setHashMember(obj, "query", &object.Builtin{Name: "db.tx.query", Fn: dbTxQuery, Extra: &object.GoObject{Value: tx}})
	setHashMember(obj, "queryOne", &object.Builtin{Name: "db.tx.queryOne", Fn: dbTxQueryOne, Extra: &object.GoObject{Value: tx}})
	setHashMember(obj, "prepare", &object.Builtin{Name: "db.tx.prepare", Fn: dbTxPrepare, Extra: &object.GoObject{Value: tx}})
	setHashMember(obj, "commit", &object.Builtin{Name: "db.tx.commit", Fn: dbTxCommit, Extra: &object.GoObject{Value: tx}})
	setHashMember(obj, "rollback", &object.Builtin{Name: "db.tx.rollback", Fn: dbTxRollback, Extra: &object.GoObject{Value: tx}})
	return obj
}

func dbTxExec(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	tx, errObj := boundTx(pos, env, "db.tx.exec")
	if errObj != nil {
		return errObj
	}
	query, params, errObj := dbQueryArgs(pos, "db.tx.exec", args)
	if errObj != nil {
		return errObj
	}
	result, err := tx.tx.Exec(query, params...)
	if err != nil {
		return object.NewError(pos, "db.tx.exec: %v", err)
	}
	return sqlResultObject(result)
}

func dbTxQuery(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	tx, errObj := boundTx(pos, env, "db.tx.query")
	if errObj != nil {
		return errObj
	}
	query, params, errObj := dbQueryArgs(pos, "db.tx.query", args)
	if errObj != nil {
		return errObj
	}
	return queryWithExecutor(pos, "db.tx.query", tx.tx, query, params)
}

func dbTxQueryOne(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	result := dbTxQuery(env, pos, args...)
	return firstRow(result)
}

func dbTxPrepare(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	tx, errObj := boundTx(pos, env, "db.tx.prepare")
	if errObj != nil {
		return errObj
	}
	query, errObj := requiredString(pos, "db.tx.prepare", args, 0, "query")
	if errObj != nil {
		return errObj
	}
	stmt, err := tx.tx.Prepare(query)
	if err != nil {
		return object.NewError(pos, "db.tx.prepare: %v", err)
	}
	return dbStmtObject(&dbStmt{stmt: stmt})
}

func dbTxCommit(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	tx, errObj := boundTx(pos, env, "db.tx.commit")
	if errObj != nil {
		return errObj
	}
	if err := tx.tx.Commit(); err != nil {
		return object.NewError(pos, "db.tx.commit: %v", err)
	}
	return object.UNDEFINED
}

func dbTxRollback(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	tx, errObj := boundTx(pos, env, "db.tx.rollback")
	if errObj != nil {
		return errObj
	}
	if err := tx.tx.Rollback(); err != nil {
		return object.NewError(pos, "db.tx.rollback: %v", err)
	}
	return object.UNDEFINED
}

func dbStmtObject(stmt *dbStmt) *object.Hash {
	obj := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
	setHashMember(obj, "__stmt", &object.GoObject{Value: stmt})
	setHashMember(obj, "exec", &object.Builtin{Name: "db.stmt.exec", Fn: dbStmtExec, Extra: &object.GoObject{Value: stmt}})
	setHashMember(obj, "query", &object.Builtin{Name: "db.stmt.query", Fn: dbStmtQuery, Extra: &object.GoObject{Value: stmt}})
	setHashMember(obj, "queryOne", &object.Builtin{Name: "db.stmt.queryOne", Fn: dbStmtQueryOne, Extra: &object.GoObject{Value: stmt}})
	setHashMember(obj, "close", &object.Builtin{Name: "db.stmt.close", Fn: dbStmtClose, Extra: &object.GoObject{Value: stmt}})
	return obj
}

func dbStmtExec(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	stmt, errObj := boundStmt(pos, env, "db.stmt.exec")
	if errObj != nil {
		return errObj
	}
	params := dbParams(args)
	result, err := stmt.stmt.Exec(params...)
	if err != nil {
		return object.NewError(pos, "db.stmt.exec: %v", err)
	}
	return sqlResultObject(result)
}

func dbStmtQuery(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	stmt, errObj := boundStmt(pos, env, "db.stmt.query")
	if errObj != nil {
		return errObj
	}
	params := dbParams(args)
	rows, err := stmt.stmt.Query(params...)
	if err != nil {
		return object.NewError(pos, "db.stmt.query: %v", err)
	}
	defer rows.Close()
	result, err := rowsToArray(rows)
	if err != nil {
		return object.NewError(pos, "db.stmt.query: %v", err)
	}
	return result
}

func dbStmtQueryOne(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	result := dbStmtQuery(env, pos, args...)
	return firstRow(result)
}

func dbStmtClose(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	stmt, errObj := boundStmt(pos, env, "db.stmt.close")
	if errObj != nil {
		return errObj
	}
	if err := stmt.stmt.Close(); err != nil {
		return object.NewError(pos, "db.stmt.close: %v", err)
	}
	return object.UNDEFINED
}

func dbQueryArgs(pos ast.Position, name string, args []object.Object) (string, []interface{}, *object.Error) {
	query, errObj := requiredString(pos, name, args, 0, "query")
	if errObj != nil {
		return "", nil, errObj
	}
	params := make([]interface{}, 0)
	if len(args) >= 2 {
		if arr, ok := args[1].(*object.Array); ok {
			for _, item := range arr.Elements {
				params = append(params, objectToSQLValue(item))
			}
		} else {
			for _, item := range args[1:] {
				params = append(params, objectToSQLValue(item))
			}
		}
	}
	return query, params, nil
}

func dbParams(args []object.Object) []interface{} {
	params := make([]interface{}, 0)
	if len(args) == 1 {
		if arr, ok := args[0].(*object.Array); ok {
			for _, item := range arr.Elements {
				params = append(params, objectToSQLValue(item))
			}
			return params
		}
	}
	for _, item := range args {
		params = append(params, objectToSQLValue(item))
	}
	return params
}

func queryWithExecutor(pos ast.Position, name string, executor dbExecutor, query string, params []interface{}) object.Object {
	rows, err := executor.Query(query, params...)
	if err != nil {
		return object.NewError(pos, "%s: %v", name, err)
	}
	defer rows.Close()
	result, err := rowsToArray(rows)
	if err != nil {
		return object.NewError(pos, "%s: %v", name, err)
	}
	return result
}

func firstRow(result object.Object) object.Object {
	if object.IsRuntimeError(result) {
		return result
	}
	rows, ok := result.(*object.Array)
	if !ok || len(rows.Elements) == 0 {
		return object.NULL
	}
	return rows.Elements[0]
}

func sqlResultObject(result sql.Result) *object.Hash {
	out := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
	if rows, err := result.RowsAffected(); err == nil {
		setHashMember(out, "rowsAffected", &object.Number{Value: float64(rows)})
	}
	if id, err := result.LastInsertId(); err == nil {
		setHashMember(out, "lastInsertId", &object.Number{Value: float64(id)})
	}
	return out
}

func rowsToArray(rows *sql.Rows) (*object.Array, error) {
	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	result := []object.Object{}
	for rows.Next() {
		values := make([]interface{}, len(cols))
		ptrs := make([]interface{}, len(cols))
		for i := range values {
			ptrs[i] = &values[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return nil, err
		}
		row := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
		for i, col := range cols {
			setHashMember(row, col, sqlValueToObject(values[i]))
		}
		result = append(result, row)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return &object.Array{Elements: result}, nil
}

func objectToSQLValue(obj object.Object) interface{} {
	switch v := obj.(type) {
	case *object.Null, *object.Undefined:
		return nil
	case *object.String:
		return v.Value
	case *object.Number:
		if math.Trunc(v.Value) == v.Value {
			return int64(v.Value)
		}
		return v.Value
	case *object.Boolean:
		return v.Value
	default:
		return obj.Inspect()
	}
}

func sqlValueToObject(value interface{}) object.Object {
	switch v := value.(type) {
	case nil:
		return object.NULL
	case []byte:
		return &object.String{Value: string(v)}
	case string:
		return &object.String{Value: v}
	case bool:
		return object.NativeBool(v)
	case int:
		return &object.Number{Value: float64(v)}
	case int32:
		return &object.Number{Value: float64(v)}
	case int64:
		return &object.Number{Value: float64(v)}
	case float32:
		return &object.Number{Value: float64(v)}
	case float64:
		return &object.Number{Value: v}
	default:
		return &object.String{Value: fmt.Sprint(v)}
	}
}

func boundDB(pos ast.Position, env *object.Environment, name string) (*dbConn, *object.Error) {
	goObj, ok := env.Extra.(*object.GoObject)
	if !ok {
		return nil, object.NewError(pos, "%s: missing db receiver", name)
	}
	conn, ok := goObj.Value.(*dbConn)
	if !ok {
		return nil, object.NewError(pos, "%s: invalid db receiver", name)
	}
	return conn, nil
}

func boundTx(pos ast.Position, env *object.Environment, name string) (*dbTx, *object.Error) {
	goObj, ok := env.Extra.(*object.GoObject)
	if !ok {
		return nil, object.NewError(pos, "%s: missing tx receiver", name)
	}
	tx, ok := goObj.Value.(*dbTx)
	if !ok {
		return nil, object.NewError(pos, "%s: invalid tx receiver", name)
	}
	return tx, nil
}

func boundStmt(pos ast.Position, env *object.Environment, name string) (*dbStmt, *object.Error) {
	goObj, ok := env.Extra.(*object.GoObject)
	if !ok {
		return nil, object.NewError(pos, "%s: missing stmt receiver", name)
	}
	stmt, ok := goObj.Value.(*dbStmt)
	if !ok {
		return nil, object.NewError(pos, "%s: invalid stmt receiver", name)
	}
	return stmt, nil
}

func requiredInt(pos ast.Position, name string, args []object.Object, index int, label string) (int, *object.Error) {
	if len(args) <= index {
		return 0, object.NewError(pos, "%s requires %s", name, label)
	}
	n, ok := args[index].(*object.Number)
	if !ok {
		return 0, object.NewError(pos, "%s: %s must be a number", name, label)
	}
	return int(n.Value), nil
}

func normalizeDBDriver(driver string) (string, error) {
	switch strings.ToLower(driver) {
	case "sqlite", "sqlite3":
		return "sqlite", nil
	case "postgres", "postgresql", "pg":
		return "postgres", nil
	case "mysql":
		return "mysql", nil
	case "mssql", "sqlserver", "sql-server":
		return "sqlserver", nil
	default:
		return "", fmt.Errorf("unsupported driver %q", driver)
	}
}
