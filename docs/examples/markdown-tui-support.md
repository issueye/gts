# Markdown / TUI 标准库脚本示例

这些示例展示脚本侧如何组合 `@std/text`、`@std/markdown`、`@std/highlight`、`@std/terminal` 和 `@std/table`。它们适合用于 agent 最终回答、流式输出预览、网页摘要、工具结果和 README/计划文档展示。

## 宽度感知文本处理

```javascript
let text = require("@std/text");

let title = "\x1b[36m你好 GoScript\x1b[0m";

println(text.width(title));                  // 13
println(text.truncateWidth(title, 8));        // 你好 Go
println(text.padRightWidth("你", 4) + "|");   // 你  |

let lines = text.wrapWidth("你好世界 agent", 8);
for (let line of lines) {
  println(line);
}
```

## Markdown 解析与终端渲染

```javascript
let markdown = require("@std/markdown");
let text = require("@std/text");

let source = "# 结果\n\n- **状态**：完成\n- [文档](https://example.com/docs)\n\n```gs\nprintln(\"ok\")\n```";

let doc = markdown.parse(source);
println(doc.children[0].type); // heading

let rendered = markdown.renderTerminal(source, {
  width: 40
});

for (let line of rendered.lines) {
  println(line);
}

for (let link of rendered.links) {
  println("link:", link.text, link.url);
}
```

## 流式 Markdown 预览

```javascript
let markdown = require("@std/markdown");

let stream = markdown.createStream({
  width: 60
});

stream.append("# 回答\n\n");
stream.append("正在生成第一段内容，");

let preview = stream.snapshot();
for (let line of preview.lines) {
  println(line);
}

stream.append("现在补齐最后内容。\n");

let final = stream.finalize();
println(final.document.type); // document
```

## HTML 转 Markdown

```javascript
let markdown = require("@std/markdown");

let html = "<h1>Title</h1><p>Hello <a href=\"/docs\">docs</a></p>";

let md = markdown.fromHTML(html, {
  baseUrl: "https://example.com",
  includeLinks: true,
  maxChars: 20000
});

println(md);
```

输出会类似：

```markdown
# Title

Hello [docs](https://example.com/docs)
```

## 代码块轻量高亮

```javascript
let highlight = require("@std/highlight");

let out = highlight.terminal("{\"ok\": true}", {
  lang: "json",
  width: 40,
  color: true
});

for (let line of out.lines) {
  println(line);
}
```

## 终端样式和 OSC 8 链接

```javascript
let terminal = require("@std/terminal");
let text = require("@std/text");

let heading = terminal.style("Search Results", {
  bold: true,
  fg: "accent"
});

println(heading);
println(text.width(heading)); // 14

let link = terminal.hyperlink("OpenAI", "https://openai.com", {
  enabled: true
});

println(link);

let fallback = terminal.hyperlink("OpenAI", "https://openai.com", {
  enabled: false
});

println(fallback); // OpenAI <https://openai.com>
```

## 表格布局

```javascript
let table = require("@std/table");

let result = table.layout([
  ["Name", "Description"],
  ["web_fetch", "读取网页并转换为 Markdown"],
  ["tool_result", "展示工具调用输出"]
], {
  width: 50,
  minColWidth: 8,
  wrap: true
});

for (let line of result.lines) {
  println(line);
}
```

## 一个简单组合渲染函数

```javascript
let markdown = require("@std/markdown");
let terminal = require("@std/terminal");

function renderAgentAnswer(source, width) {
  let out = markdown.renderTerminal(source, {
    width: width
  });

  for (let line of out.lines) {
    println(line);
  }

  if (out.links.length > 0) {
    println("");
    println(terminal.style("Links", { bold: true, fg: "muted" }));
    for (let item of out.links) {
      println("- " + terminal.hyperlink(item.text, item.url, { enabled: true }));
    }
  }
}

renderAgentAnswer("# Done\n\nSee [guide](https://example.com/guide).", 80);
```
