requireRole("hub_user");
let allRows = [];
const tbody = document.getElementById("rows");
function render(rows) { setRows(tbody, rows, [{ key: "invoice_id" }, { key: "vendor_name" }, { key: "warehouse_id" }, { key: "status" }]); }
async function load() { allRows = await API.invoices(); render(allRows); }
wireSearch(document.getElementById("search"), () => allRows, render);
load().catch((e) => toast(e.message, true));
