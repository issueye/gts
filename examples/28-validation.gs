// ============================================================
// 28-validation.gs -- 数据验证示例
// ============================================================

let v = require("@std/validation");

function main() {
  console.log("=== @std/validation - 数据验证示例 ===\n");

  // 1. 字符串验证
  console.log("1. 字符串验证");
  testStringValidation();

  // 2. 数字验证
  console.log("\n2. 数字验证");
  testNumberValidation();

  // 3. 数组验证
  console.log("\n3. 数组验证");
  testArrayValidation();

  // 4. 链式 API
  console.log("\n4. 链式 API");
  testChaining();

  // 5. required vs optional
  console.log("\n5. Required vs Optional");
  testRequiredOptional();

  // 6. 实际应用：用户注册
  console.log("\n6. 实际应用：用户注册");
  testUserRegistration();
}

function testStringValidation() {
  // 长度验证
  let minSchema = v.string().min(3);
  let result1 = minSchema.validate("ab");
  console.log("  min(3) on 'ab':", result1.valid, result1.error || "");

  let result2 = minSchema.validate("hello");
  console.log("  min(3) on 'hello':", result2.valid);

  // 邮箱验证
  let emailSchema = v.string().email();
  let result3 = emailSchema.validate("test@example.com");
  console.log("  email on 'test@example.com':", result3.valid);

  let result4 = emailSchema.validate("invalid-email");
  console.log("  email on 'invalid-email':", result4.valid, result4.error || "");

  // URL 验证
  let urlSchema = v.string().url();
  let result5 = urlSchema.validate("https://example.com");
  console.log("  url on 'https://example.com':", result5.valid);

  // UUID 验证
  let uuidSchema = v.string().uuid();
  let result6 = uuidSchema.validate("123e4567-e89b-12d3-a456-426614174000");
  console.log("  uuid on valid UUID:", result6.valid);

  // 正则匹配
  let patternSchema = v.string().matches(/^[0-9]+$/);
  let result7 = patternSchema.validate("12345");
  console.log("  matches(/^[0-9]+$/) on '12345':", result7.valid);

  let result8 = patternSchema.validate("abc123");
  console.log("  matches(/^[0-9]+$/) on 'abc123':", result8.valid);
}

function testNumberValidation() {
  // 范围验证
  let rangeSchema = v.number().min(0).max(100);
  let result1 = rangeSchema.validate(50);
  console.log("  range [0-100] on 50:", result1.valid);

  let result2 = rangeSchema.validate(150);
  console.log("  range [0-100] on 150:", result2.valid, result2.error || "");

  // 整数验证
  let intSchema = v.number().int();
  let result3 = intSchema.validate(42);
  console.log("  int on 42:", result3.valid);

  let result4 = intSchema.validate(3.14);
  console.log("  int on 3.14:", result4.valid, result4.error || "");

  // 正数验证
  let positiveSchema = v.number().positive();
  let result5 = positiveSchema.validate(5);
  console.log("  positive on 5:", result5.valid);

  let result6 = positiveSchema.validate(-1);
  console.log("  positive on -1:", result6.valid, result6.error || "");
}

function testArrayValidation() {
  let arraySchema = v.array().min(1).max(5);

  let result1 = arraySchema.validate([1, 2, 3]);
  console.log("  array [1-5] on [1,2,3]:", result1.valid);

  let result2 = arraySchema.validate([]);
  console.log("  array [1-5] on []:", result2.valid, result2.error || "");

  let result3 = arraySchema.validate([1, 2, 3, 4, 5, 6]);
  console.log("  array [1-5] on [1..6]:", result3.valid, result3.error || "");
}

function testChaining() {
  // 多个验证规则链式调用
  let schema = v.string()
    .min(3)
    .max(20)
    .matches(/^[a-zA-Z0-9_]+$/)
    .required();

  let result1 = schema.validate("john_doe");
  console.log("  username 'john_doe':", result1.valid);

  let result2 = schema.validate("ab");
  console.log("  username 'ab':", result2.valid, result2.error || "");

  let result3 = schema.validate("user@name");
  console.log("  username 'user@name':", result3.valid, result3.error || "");
}

function testRequiredOptional() {
  // Required
  let requiredSchema = v.string().required();
  let result1 = requiredSchema.validate(undefined);
  console.log("  required on undefined:", result1.valid, result1.error || "");

  let result2 = requiredSchema.validate("value");
  console.log("  required on 'value':", result2.valid);

  // Optional
  let optionalSchema = v.string().min(3).optional();
  let result3 = optionalSchema.validate(undefined);
  console.log("  optional on undefined:", result3.valid);

  let result4 = optionalSchema.validate("ab");
  console.log("  optional on 'ab':", result4.valid, result4.error || "");

  let result5 = optionalSchema.validate("hello");
  console.log("  optional on 'hello':", result5.valid);
}

function testUserRegistration() {
  // 定义验证规则
  let usernameSchema = v.string()
    .min(3)
    .max(20)
    .matches(/^[a-zA-Z0-9_]+$/)
    .required();

  let emailSchema = v.string()
    .email()
    .required();

  let ageSchema = v.number()
    .int()
    .min(0)
    .max(150)
    .optional();

  // 验证函数
  function validateUser(data) {
    // 使用 parse() 方法 - 失败会抛出异常
    try {
      let username = usernameSchema.parse(data.username);
      let email = emailSchema.parse(data.email);
      let age = ageSchema.parse(data.age);

      return {
        valid: true,
        user: {
          username: username,
          email: email,
          age: age
        }
      };
    } catch (e) {
      return {
        valid: false,
        error: e
      };
    }
  }

  // 测试用例 1：有效数据
  let result1 = validateUser({
    username: "john_doe",
    email: "john@example.com",
    age: 25
  });
  console.log("  Valid user:", result1.valid);
  if (result1.valid) {
    console.log("    Username:", result1.user.username);
    console.log("    Email:", result1.user.email);
    console.log("    Age:", result1.user.age);
  }

  // 测试用例 2：用户名太短
  let result2 = validateUser({
    username: "ab",
    email: "test@example.com"
  });
  console.log("\n  Invalid username:", result2.valid);
  if (!result2.valid) {
    console.log("    Error:", result2.error);
  }

  // 测试用例 3：邮箱格式错误
  let result3 = validateUser({
    username: "john_doe",
    email: "invalid-email"
  });
  console.log("\n  Invalid email:", result3.valid);
  if (!result3.valid) {
    console.log("    Error:", result3.error);
  }

  // 测试用例 4：可选年龄未提供
  let result4 = validateUser({
    username: "jane_doe",
    email: "jane@example.com"
  });
  console.log("\n  Optional age omitted:", result4.valid);
  if (result4.valid) {
    console.log("    Age:", result4.user.age);
  }
}

main();
