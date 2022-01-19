function main(args, out, res) {
  var array1 = [0, 1, 2, 3];
  var array2 = [{ name: "test" }, { name: "ping", sort: 10 }];
  var array3 = [0, { name: "test" }, 2];

  var data = args[0] || [];
  for (var i in data) {
    data[i]["array1"] = array1;
    data[i]["array2"] = array2;
    data[i]["array3"] = array3;
    data[i]["array4"] = [0, 1, 2, 3];
  }

  return data;
}
