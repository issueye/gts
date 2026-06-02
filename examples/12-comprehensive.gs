// ============================================================
// 12-comprehensive.gs —— 综合实战：图书管理系统
// ============================================================
// 综合运用之前学到的所有知识

// --- 定义类型 ---
type Book = {
  id: number,
  title: string,
  author: string,
  year: number,
  status: string,   // "available" | "borrowed"
};

// --- 图书馆类 ---
class Library {
  books: Book[];
  nextId: number;

  constructor() {
    this.books = [];
    this.nextId = 1;
  }

  // 添加图书
  addBook(title: string, author: string, year: number): Book {
    let book: Book = {
      id: this.nextId,
      title: title,
      author: author,
      year: year,
      status: "available",
    };
    this.nextId = this.nextId + 1;
    this.books.push(book);
    console.log(`  已添加: 《${title}》 (ID: ${book.id})`);
    return book;
  }

  // 借阅图书
  borrowBook(id: number): boolean {
    let book = this.findById(id);
    if (book === null) {
      console.log(`  错误: ID ${id} 的图书不存在`);
      return false;
    }
    if (book.status === "borrowed") {
      console.log(`  错误: 《${book.title}》已被借出`);
      return false;
    }
    book.status = "borrowed";
    console.log(`  借阅成功: 《${book.title}》`);
    return true;
  }

  // 归还图书
  returnBook(id: number): boolean {
    let book = this.findById(id);
    if (book === null) {
      console.log(`  错误: ID ${id} 的图书不存在`);
      return false;
    }
    if (book.status === "available") {
      console.log(`  提示: 《${book.title}》已在馆中`);
      return false;
    }
    book.status = "available";
    console.log(`  归还成功: 《${book.title}》`);
    return true;
  }

  // 按 ID 查找
  findById(id: number): Book | null {
    for (let b of this.books) {
      if (b.id === id) { return b; }
    }
    return null;
  }

  // 搜索（支持标题和作者模糊匹配）
  search(keyword: string): Book[] {
    let lower = keyword.toLowerCase();
    return this.books.filter(b =>
      b.title.toLowerCase().includes(lower) ||
      b.author.toLowerCase().includes(lower)
    );
  }

  // 获取所有在馆图书
  getAvailable(): Book[] {
    return this.books.filter(b => b.status === "available");
  }

  // 获取已借出图书
  getBorrowed(): Book[] {
    return this.books.filter(b => b.status === "borrowed");
  }

  // 统计信息
  stats() {
    let total = this.books.length;
    let available = this.getAvailable().length;
    let borrowed = this.getBorrowed().length;

    // 按年份分组统计
    let byYear = {};
    for (let b of this.books) {
      let y = b.year;
      if (byYear[y] === undefined) {
        byYear[y] = 0;
      }
      byYear[y] = byYear[y] + 1;
    }

    console.log("  === 图书馆统计 ===");
    console.log(`  总藏书: ${total}`);
    console.log(`  在馆: ${available}`);
    console.log(`  已借出: ${borrowed}`);
    console.log("  按年份分布:");
    for (let year in byYear) {
      console.log(`    ${year}年: ${byYear[year]} 本`);
    }
  }

  // 列出所有图书
  listAll(): void {
    if (this.books.length === 0) {
      console.log("  图书馆暂无藏书");
      return;
    }
    console.log("  === 全部藏书 ===");
    for (let b of this.books) {
      let statusIcon = match b.status {
        "available" => "[在馆]",
        "borrowed" => "[借出]",
        _ => "[未知]",
      };
      console.log(`  ${statusIcon} ${b.id}: 《${b.title}》- ${b.author} (${b.year})`);
    }
  }
}

// --- 主程序 ---
function main() {
  console.log("=== GoScript 图书管理系统 ===");
  console.log("");

  let lib = new Library();

  // 添加图书
  console.log("--- 入库 ---");
  lib.addBook("三体", "刘慈欣", 2008);
  lib.addBook("活着", "余华", 1993);
  lib.addBook("百年孤独", "马尔克斯", 1967);
  lib.addBook("红楼梦", "曹雪芹", 1791);
  lib.addBook("1984", "乔治·奥威尔", 1949);
  lib.addBook("三体2：黑暗森林", "刘慈欣", 2008);

  // 借阅操作
  console.log("");
  console.log("--- 借阅 ---");
  lib.borrowBook(1);   // 借《三体》
  lib.borrowBook(3);   // 借《百年孤独》
  lib.borrowBook(3);   // 重复借（应失败）
  lib.borrowBook(99);  // 不存在的ID（应失败）

  // 搜索
  console.log("");
  console.log("--- 搜索：'三体' ---");
  let found = lib.search("三体");
  for (let b of found) {
    console.log(`  找到: 《${b.title}》 (ID: ${b.id}, ${b.status})`);
  }

  // 在馆图书
  console.log("");
  console.log("--- 在馆图书 ---");
  let available = lib.getAvailable();
  for (let b of available) {
    console.log(`  ${b.id}: 《${b.title}》`);
  }

  // 归还
  console.log("");
  console.log("--- 归还 ---");
  lib.returnBook(1);   // 归还《三体》

  // 列出全部
  console.log("");
  lib.listAll();

  // 统计
  console.log("");
  lib.stats();
}

main();
