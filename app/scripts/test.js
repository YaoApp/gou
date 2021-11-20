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

function helloProcess(name) {
  out = Process("plugins.user.Login", 1024, name);
  return {
    out: out,
    name: name,
  };
}

function helloGlobal(name, global) {
  return "hello:" + name + ",global:" + global.hello;
}

function helloSession(value) {
  Process("session.Set", "foo", value);
  out = Process("session.Get", "foo");
  return {
    input: value,
    out: out,
  };
}
