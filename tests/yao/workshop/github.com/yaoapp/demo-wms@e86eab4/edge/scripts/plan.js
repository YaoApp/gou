/**
 * Update Items
 */
function AfterSave(id, payload) {
  console.log(id, payload);
  var items = payload.items || {};
  var deletes = items.delete || [];
  var data = items.data || [];
  if (data.length > 0 || deletes.length > 0) {
    for (var i in data) {
      delete data[i].amount;
    }

    // Save
    var res = Process("models.plan.item.EachSaveAfterDelete", deletes, data, {
      plan_id: id,
    });
    if (res.code && res.code > 300) {
      console.log("Plan:AfterSave Error:", res);
      return id;
    }
  }
  return id;
}
