requireRole("central_admin");
const tbody = document.getElementById("rows");
function params(format) {
  const p = new URLSearchParams();
  ["from", "to", "warehouse", "vendor", "status"].forEach((id) => {
    const val = document.getElementById(id).value.trim();
    if (val) p.set(id === "warehouse" ? "warehouse_id" : id, val);
  });
  if (format) p.set("format", format);
  return "?" + p.toString();
}
async function load() {
  const rows = await API.reports(params());
  setRows(tbody, rows, [{ key: "invoice" }, { key: "vendor" }, { key: "warehouse" }, { key: "status" }, { key: "sku" }, { key: "item" }, { key: "expected" }, { key: "received" }, { key: "variance" }]);
}
document.getElementById("filter").onclick = load;
document.getElementById("csv").onclick = () => API.request("/api/admin/reports/reconciliation" + params()).then((r) => downloadBlob(r, "reconciliation_report.csv"));
document.getElementById("excel").onclick = () => API.request("/api/admin/reports/reconciliation" + params("xlsx")).then((r) => downloadBlob(r, "reconciliation_report.xls"));
load().catch((e) => toast(e.message, true));
