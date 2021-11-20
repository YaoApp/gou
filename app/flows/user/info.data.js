function main(args, out, res, global) {
  session = Process("session.dump");
  return {
    global: global,
    session: session,
  };
}
