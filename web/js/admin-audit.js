requireRole("central_admin");
let allRows = [];
const tbody = document.getElementById("rows");
function render(rows) {
  setRows(tbody, rows, [{ key: "timestamp" }, { key: "invoice" }, { key: "sku" }, { key: "warehouse" }, { key: "user" }, { key: "action" }, { key: "old_quantity" }, { key: "new_quantity" }, { key: "reason" }]);
}
async function load() { allRows = await API.audit(); render(allRows); }
wireSearch(document.getElementById("search"), () => allRows, render);
load().catch((e) => toast(e.message, true));
