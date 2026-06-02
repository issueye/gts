let db = require("@std/db");

let conn = db.open("sqlite", ":memory:");

conn.exec("create table tasks (id integer primary key, title text, done integer)");
let insert = conn.prepare("insert into tasks (title, done) values (?, ?)");
insert.exec(["read docs", 1]);
insert.exec(["write agent", 0]);
insert.close();

let tx = conn.begin();
tx.exec("insert into tasks (title, done) values (?, ?)", ["rolled back", 1]);
tx.rollback();

let rows = conn.query("select title from tasks where done = ? order by id", [1]);
let first = conn.queryOne("select title from tasks where id = ?", [2]);

conn.close();

println(rows[0].title);
println(first.title);
