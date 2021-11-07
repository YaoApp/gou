function main(args, out, res) {
  return {
    name: "rank",
    args: args,
    out: out,
    res: res,
    hello: hello(),
  };
}

function hello() {
  return "rank hello";
}
