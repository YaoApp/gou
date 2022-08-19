const getTime = () => {
  return new Date().toISOString();
};

function now() {
  console.log("====", __yao_global, __yao_sid, "===");
  UnitTestFn("UnitTestFn Output", "1", 2, 0.618, { foo: "bar" }, [
    "1",
    2,
    0.618,
    { hello: "world" },
  ]);
  return getTime();
}

function promiseTest() {
  return new Promise((resole, reject) => {
    resole(true);
  });
}

async function asyncTest() {
  const body = await new Promise((resole, reject) => {
    resole(JSON.stringify({ foo: bar }));
  })();
  const data = JSON.parse(body);
  console.log(data);
  return data;
}

function processTest() {
  Process("flows.test", "hello", 1, 0.618, { foo: "bar" }, [
    "world",
    1,
    0.618,
    "test",
  ]);
}
