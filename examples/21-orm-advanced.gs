// ORM 高级示例 - 实际应用场景
import orm from "@std/orm";

// 连接到 MySQL 数据库
const db = orm.connect("mysql", "root:password@tcp(localhost:3306)/myapp");

// === 用户管理系统 ===

// 创建新用户
function createUser(name, email, password) {
    const result = db.table("users").insert({
        name: name,
        email: email,
        password: password,
        created_at: Date.now()
    });

    return result.lastInsertId;
}

// 根据邮箱查找用户
function findUserByEmail(email) {
    return db.table("users")
        .where("email = ?", email)
        .first();
}

// 更新用户信息
function updateUser(userId, data) {
    return db.table("users")
        .where("id = ?", userId)
        .update(data);
}

// 删除用户
function deleteUser(userId) {
    return db.table("users")
        .where("id = ?", userId)
        .delete();
}

// 获取活跃用户列表（分页）
function getActiveUsers(page, pageSize) {
    const offset = (page - 1) * pageSize;

    return db.table("users")
        .where("status = ?", "active")
        .orderBy("last_login DESC")
        .limit(pageSize)
        .offset(offset)
        .find();
}

// 搜索用户
function searchUsers(keyword) {
    return db.table("users")
        .where("name LIKE ?", "%" + keyword + "%")
        .where("status = ?", "active")
        .orderBy("name ASC")
        .find();
}

// === 文章管理系统 ===

// 创建文章
function createPost(title, content, authorId) {
    const tx = db.begin();

    try {
        const result = tx.table("posts").insert({
            title: title,
            content: content,
            author_id: authorId,
            status: "draft",
            created_at: Date.now()
        });

        // 更新作者的文章计数
        tx.table("users")
            .where("id = ?", authorId)
            .update({
                post_count: db.table("posts")
                    .where("author_id = ?", authorId)
                    .count() + 1
            });

        tx.commit();
        return result.lastInsertId;
    } catch (err) {
        tx.rollback();
        throw err;
    }
}

// 获取已发布的文章列表
function getPublishedPosts(page, pageSize) {
    const offset = (page - 1) * pageSize;

    return db.table("posts")
        .where("status = ?", "published")
        .orderBy("created_at DESC")
        .limit(pageSize)
        .offset(offset)
        .find();
}

// 获取用户的所有文章
function getUserPosts(userId) {
    return db.table("posts")
        .where("author_id = ?", userId)
        .orderBy("created_at DESC")
        .find();
}

// 发布文章
function publishPost(postId) {
    return db.table("posts")
        .where("id = ?", postId)
        .update({
            status: "published",
            published_at: Date.now()
        });
}

// === 评论系统 ===

// 添加评论
function addComment(postId, userId, content) {
    return db.table("comments").insert({
        post_id: postId,
        user_id: userId,
        content: content,
        created_at: Date.now()
    });
}

// 获取文章的评论
function getPostComments(postId) {
    return db.table("comments")
        .where("post_id = ?", postId)
        .orderBy("created_at DESC")
        .find();
}

// 删除评论
function deleteComment(commentId, userId) {
    return db.table("comments")
        .where("id = ?", commentId)
        .where("user_id = ?", userId)
        .delete();
}

// === 统计分析 ===

// 获取用户统计
function getUserStats() {
    const total = db.table("users").count();

    const active = db.table("users")
        .where("status = ?", "active")
        .count();

    const newThisMonth = db.table("users")
        .where("created_at >= ?", getMonthStart())
        .count();

    return {
        total: total,
        active: active,
        newThisMonth: newThisMonth
    };
}

// 获取文章统计
function getPostStats() {
    const total = db.table("posts").count();

    const published = db.table("posts")
        .where("status = ?", "published")
        .count();

    const draft = db.table("posts")
        .where("status = ?", "draft")
        .count();

    return {
        total: total,
        published: published,
        draft: draft
    };
}

// === 批量操作 ===

// 批量创建用户
function batchCreateUsers(users) {
    const tx = db.begin();

    try {
        for (const user of users) {
            tx.table("users").insert(user);
        }
        tx.commit();
        return true;
    } catch (err) {
        tx.rollback();
        throw err;
    }
}

// 批量更新状态
function batchUpdateStatus(userIds, status) {
    const tx = db.begin();

    try {
        for (const id of userIds) {
            tx.table("users")
                .where("id = ?", id)
                .update({ status: status });
        }
        tx.commit();
        return true;
    } catch (err) {
        tx.rollback();
        throw err;
    }
}

// === 辅助函数 ===

function getMonthStart() {
    const now = new Date();
    return new Date(now.getFullYear(), now.getMonth(), 1).getTime();
}

// === 使用示例 ===

// 创建用户
const userId = createUser("张三", "zhangsan@example.com", "password123");
console.log("创建用户 ID:", userId);

// 查找用户
const user = findUserByEmail("zhangsan@example.com");
console.log("找到用户:", user);

// 创建文章
const postId = createPost("我的第一篇文章", "这是内容...", userId);
console.log("创建文章 ID:", postId);

// 发布文章
publishPost(postId);
console.log("文章已发布");

// 获取统计
const stats = getUserStats();
console.log("用户统计:", stats);

// 关闭连接
db.close();
