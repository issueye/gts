// fetch.gs —— 用 fetch 异步请求并解析 JSON
async function getUser(id: number) {
  let resp = await fetch(`https://jsonplaceholder.typicode.com/users/${id}`);
  if (!resp.ok) {
    throw new Error(`HTTP ${resp.status}`);
  }
  return await resp.json();
}

async function main() {
  try {
    let user = await getUser(1);
    console.log(`${user.name} <${user.email}>`);
  } catch (e) {
    console.error("failed:", e.message);
  }
}

main();
