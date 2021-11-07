function main(args, out, res) {
  return {
    name: "sort",
    args: args,
    out: out,
    res: res,
    hello: hello(),
  };
}

function hello() {
  return "sort hello";
}
