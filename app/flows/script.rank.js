function main(args, out, res) {
  return {
    name: "rank",
    args: args,
    out: out,
    res: res,
    hello: hello(),
    user: user(),
  };
}

function hello() {
  return "rank hello";
}

function user() {
  return Process("plugins.user.Login", 1024);
}
