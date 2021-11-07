function main(args) {
  return {
    args: args,
    now: now(),
    lastYear: lastYear(),
    hello: hello(args[0]),
  };
}

function now() {
  return new Date().toISOString().split("T")[0];
}

function lastYear() {
  var d = new Date();
  d.setFullYear(d.getFullYear() - 1);
  return d.toISOString().split("T")[0];
}

function hello(name) {
  return "hello:" + name;
}
