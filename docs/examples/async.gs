// async.gs —— async/await、Promise、setTimeout
function delay(ms: number): Promise<undefined> {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

async function main() {
  console.log("start");
  await delay(100);
  console.log("after 100ms");

  let values: number[] = await Promise.all([
    Promise.resolve(1),
    Promise.resolve(2),
    Promise.resolve(3),
  ]);
  console.log("values =", values);

  try {
    await Promise.reject(new Error("oops"));
  } catch (e) {
    console.error("caught:", e.message);
  }

  console.log("done");
}

main();
