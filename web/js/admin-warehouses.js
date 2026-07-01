requireRole("central_admin");
let allRows = [];
const tbody = document.getElementById("rows");
function render(rows) {
  setRows(tbody, rows, [
    { key: "id" }, { key: "name" }, { key: "user_count" },
    { raw: true, render: (r) => `<button class="small-btn secondary" data-edit="${r.id}">Rename</button> <button class="small-btn danger-btn" data-delete="${r.id}">Delete</button>` },
  ]);
}
async function load() { allRows = await API.warehouses(); render(allRows); }
document.getElementById("createWh").onclick = async () => {
  try {
    await API.createWarehouse(document.getElementById("whId").value.trim(), document.getElementById("whName").value.trim());
    toast("Warehouse created");
    await load();
  } catch (e) { toast(e.message, true); }
};
tbody.onclick = async (e) => {
  const edit = e.target.dataset.edit;
  const del = e.target.dataset.delete;
  try {
    if (edit) {
      const name = prompt("Warehouse name");
      if (name) await API.updateWarehouse(edit, name);
    }
    if (del && confirm("Delete warehouse?")) await API.deleteWarehouse(del);
    await load();
  } catch (err) { toast(err.message, true); }
};
wireSearch(document.getElementById("search"), () => allRows, render);
load().catch((e) => toast(e.message, true));
