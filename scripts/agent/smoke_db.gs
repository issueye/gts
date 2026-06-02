let db = require("@std/db");

let conn = db.open("sqlite", ":memory:");
conn.exec("create table notes (id integer primary key, title text)");
conn.exec("insert into notes (title) values (?)", ["agent-db-ok"]);

let row = conn.queryOne("select title from notes where id = ?", [1]);
conn.close();

println(row.title);
