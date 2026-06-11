import db from "@std/db";

function quote(driver, name) {
  if (driver === "mysql") {
    return "`" + name + "`";
  }
  if (driver === "sqlserver") {
    return "[" + name + "]";
  }
  return "\"" + name + "\"";
}

function placeholder(driver, n) {
  if (driver === "postgres") {
    return "$" + String(n);
  }
  return "?";
}

function quotedFields(driver, fields) {
  return fields.map(field => quote(driver, field)).join(", ");
}

function columnDefinition(column) {
  if (typeof column === "string") {
    return column;
  }
  let parts = [quote("", column.name), column.type || "text"];
  if (column.primaryKey) {
    parts.push("primary key");
  }
  if (column.autoIncrement) {
    parts.push("autoincrement");
  }
  if (column.notNull) {
    parts.push("not null");
  }
  if (column.unique) {
    parts.push("unique");
  }
  if (column.defaultValue !== undefined) {
    parts.push("default " + String(column.defaultValue));
  }
  return parts.join(" ");
}

function columnName(column) {
  if (typeof column === "string") {
    return column.split(" ")[0];
  }
  return column.name;
}

function migrateOne(conn, schema) {
  const table = schema.table || schema.name;
  if (!table) {
    throw new Error("orm.autoMigrate requires table");
  }
  const columns = schema.columns || [];
  if (columns.length === 0) {
    throw new Error("orm.autoMigrate requires columns");
  }

  const columnDefs = columns.map(column => columnDefinition(column));
  conn.exec("CREATE TABLE IF NOT EXISTS " + quote(conn.driver, table) + " (" + columnDefs.join(", ") + ")");

  if (conn.driver === "sqlite") {
    const existingRows = conn.query("PRAGMA table_info(" + quote(conn.driver, table) + ")");
    const existing = {};
    for (const row of existingRows) {
      existing[row.name] = true;
    }
    for (const column of columns) {
      const name = columnName(column);
      if (!existing[name]) {
        conn.exec("ALTER TABLE " + quote(conn.driver, table) + " ADD COLUMN " + columnDefinition(column));
      }
    }
  }

  const indexes = schema.indexes || [];
  for (const index of indexes) {
    let indexName = index.name;
    const fields = index.columns || index.fields || [];
    if (!indexName) {
      indexName = "idx_" + table + "_" + fields.join("_");
    }
    const unique = index.unique ? "UNIQUE " : "";
    conn.exec(
      "CREATE " + unique + "INDEX IF NOT EXISTS " + quote(conn.driver, indexName) +
      " ON " + quote(conn.driver, table) + " (" + quotedFields(conn.driver, fields) + ")"
    );
  }

  return true;
}

function migrateSchemas(conn, schema) {
  if (Array.isArray(schema)) {
    for (const item of schema) {
      migrateOne(conn, item);
    }
  } else {
    migrateOne(conn, schema);
  }
  return true;
}

function appendParams(target, values) {
  for (const value of values) {
    if (Array.isArray(value)) {
      for (const item of value) {
        target.push(item);
      }
    } else {
      target.push(value);
    }
  }
  return target;
}

function objectKeys(data, method) {
  const keys = Object.keys(data);
  if (keys.length === 0) {
    throw new Error(method + ": data object cannot be empty");
  }
  return keys;
}

function execInsert(conn, executor, table, data) {
  const driver = conn.driver;
  const fields = objectKeys(data, "orm.insert");
  const values = fields.map(field => data[field]);
  const placeholders = fields.map((_, i) => placeholder(driver, i + 1));
  const query = "INSERT INTO " + quote(driver, table) +
    " (" + quotedFields(driver, fields) + ") VALUES (" +
    placeholders.join(", ") + ")";
  return executor.exec(query, values);
}

function execBatchInsert(conn, executor, table, rows) {
  if (rows.length === 0) {
    return { rowsAffected: 0 };
  }

  const driver = conn.driver;
  const fields = objectKeys(rows[0], "orm.batchInsert");
  const valueSets = [];
  const values = [];
  let paramIndex = 1;

  for (const row of rows) {
    const placeholders = [];
    for (const field of fields) {
      values.push(row[field]);
      placeholders.push(placeholder(driver, paramIndex));
      paramIndex = paramIndex + 1;
    }
    valueSets.push("(" + placeholders.join(", ") + ")");
  }

  const query = "INSERT INTO " + quote(driver, table) +
    " (" + quotedFields(driver, fields) + ") VALUES " +
    valueSets.join(", ");
  return executor.exec(query, values);
}

function makeModel(conn, executor, table, fields, wheres, whereArgs, orderParts, limitValue, offsetValue) {
  const target = executor || conn;

  function clone(next) {
    return makeModel(
      conn,
      executor,
      table,
      next.fields || fields.slice(),
      next.wheres || wheres.slice(),
      next.whereArgs || whereArgs.slice(),
      next.orderParts || orderParts.slice(),
      next.limitValue === undefined ? limitValue : next.limitValue,
      next.offsetValue === undefined ? offsetValue : next.offsetValue
    );
  }

  function buildSelectQuery(countMode) {
    let selected = "*";
    if (countMode) {
      selected = "COUNT(*) AS count";
    } else if (fields.length > 0) {
      selected = quotedFields(conn.driver, fields);
    }

    let query = "SELECT " + selected + " FROM " + quote(conn.driver, table);
    if (wheres.length > 0) {
      query = query + " WHERE " + wheres.join(" AND ");
    }
    if (!countMode && orderParts.length > 0) {
      query = query + " ORDER BY " + orderParts.join(", ");
    }
    if (!countMode && limitValue > 0) {
      query = query + " LIMIT " + String(limitValue);
    }
    if (!countMode && offsetValue > 0) {
      query = query + " OFFSET " + String(offsetValue);
    }
    return query;
  }

  const model = {
    "select": function(...selectedFields) {
      return clone({ fields: selectedFields });
    },
    "where": function(condition, ...params) {
      const nextWheres = wheres.slice();
      const nextArgs = whereArgs.slice();
      nextWheres.push(condition);
      appendParams(nextArgs, params);
      return clone({ wheres: nextWheres, whereArgs: nextArgs });
    },
    "whereIn": function(field, values) {
      if (!Array.isArray(values)) {
        throw new Error("orm.whereIn: values must be an array");
      }
      if (values.length === 0) {
        throw new Error("orm.whereIn: values array cannot be empty");
      }
      const nextWheres = wheres.slice();
      const nextArgs = whereArgs.slice();
      const placeholders = [];
      for (const value of values) {
        placeholders.push(placeholder(conn.driver, nextArgs.length + 1));
        nextArgs.push(value);
      }
      nextWheres.push(quote(conn.driver, field) + " IN (" + placeholders.join(", ") + ")");
      return clone({ wheres: nextWheres, whereArgs: nextArgs });
    },
    "orderBy": function(...parts) {
      return clone({ orderParts: orderParts.concat(parts) });
    },
    "limit": function(n) {
      return clone({ limitValue: n });
    },
    "offset": function(n) {
      return clone({ offsetValue: n });
    },
    "find": function() {
      return target.query(buildSelectQuery(false), whereArgs);
    },
    "first": function() {
      const rows = clone({ limitValue: 1 }).find();
      if (rows.length === 0) {
        return null;
      }
      return rows[0];
    },
    "count": function() {
      const row = target.queryOne(buildSelectQuery(true), whereArgs);
      if (row === null) {
        return 0;
      }
      return row.count;
    },
    "insert": function(data) {
      return execInsert(conn, target, table, data);
    },
    "update": function(data) {
      const fields = objectKeys(data, "orm.update");
      const values = fields.map(field => data[field]);
      const sets = fields.map((field, i) => quote(conn.driver, field) + " = " + placeholder(conn.driver, i + 1));
      let query = "UPDATE " + quote(conn.driver, table) + " SET " + sets.join(", ");
      if (wheres.length > 0) {
        query = query + " WHERE " + wheres.join(" AND ");
      }
      return target.exec(query, values.concat(whereArgs));
    },
    "delete": function() {
      let query = "DELETE FROM " + quote(conn.driver, table);
      if (wheres.length > 0) {
        query = query + " WHERE " + wheres.join(" AND ");
      }
      return target.exec(query, whereArgs);
    },
  };
  return model;
}

function makeConnection(conn) {
  return {
    "autoMigrate": function(schema) {
      return migrateSchemas(conn, schema);
    },
    "table": function(table) {
      return makeModel(conn, null, table, [], [], [], [], 0, 0);
    },
    "insert": function(table, data) {
      return execInsert(conn, conn, table, data);
    },
    "batchInsert": function(table, rows) {
      return execBatchInsert(conn, conn, table, rows);
    },
    "begin": function() {
      const tx = conn.begin();
      return {
        "table": function(table) {
          return makeModel(conn, tx, table, [], [], [], [], 0, 0);
        },
        "commit": function() {
          return tx.commit();
        },
        "rollback": function() {
          return tx.rollback();
        },
      };
    },
    "close": function() {
      return conn.close();
    },
  };
}

export function connect(driver, dsn) {
  return makeConnection(db.open(driver, dsn));
}

export function autoMigrate(driver, dsn, schema) {
  const conn = db.open(driver, dsn);
  try {
    return migrateSchemas(conn, schema);
  } finally {
    conn.close();
  }
}

export default {
  "autoMigrate": autoMigrate,
  "connect": connect,
};
