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
  console.log(
    "user",
    "plugins.user.Login",
    Process("plugins.user.Login", 1024)
  );
  return Process("plugins.user.Login", 1024);
}
