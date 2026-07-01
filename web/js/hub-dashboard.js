requireRole("hub_user");
(async () => {
  const data = await API.hubDashboard();
  document.getElementById("cards").innerHTML = [
    ["Assigned Warehouses", data.assigned_warehouses],
    ["Pending Invoices", data.pending_invoices],
    ["Today's Received", data.today_received],
    ["Completed Today", data.completed_today],
  ].map(([label, value]) => `<section class="card"><div class="muted">${label}</div><div class="stat">${value}</div></section>`).join("");
})().catch((e) => toast(e.message, true));
