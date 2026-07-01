requireRole("central_admin");
let allRows = [];
const tbody = document.getElementById("rows");
const cols = [
  { key: "email" }, { key: "role" },
  { render: (r) => (r.warehouse_ids || []).join(", ") },
  { render: (r) => r.disabled ? "Disabled" : "Active" },
  { raw: true, render: (r) => `<button class="small-btn danger-btn" data-disable="${r.id}">Disable</button>` },
];
function render(rows) { setRows(tbody, rows, cols); }
async function load() { allRows = await API.users(); render(allRows); }
document.getElementById("saveUser").onclick = async () => {
  try {
    await API.createUser({
      email: document.getElementById("email").value.trim(),
      password: document.getElementById("password").value,
      role: document.getElementById("role").value,
      warehouse_ids: document.getElementById("warehouseIds").value.split(",").map((s) => s.trim()).filter(Boolean),
    });
    toast("User created");
    await load();
  } catch (e) { toast(e.message, true); }
};
tbody.onclick = async (e) => {
  const id = e.target.dataset.disable;
  if (!id) return;
  await API.disableUser(id);
  toast("User disabled");
  await load();
};
wireSearch(document.getElementById("search"), () => allRows, render);
load().catch((e) => toast(e.message, true));
