requireRole("central_admin");
(async () => {
  const data = await API.adminDashboard();
  const cards = [
    ["Total Warehouses", data.total_warehouses],
    ["Total Users", data.total_users],
    ["Pending", data.pending],
    ["Receiving", data.receiving],
    ["Completed", data.completed],
  ];
  document.getElementById("cards").innerHTML = cards.map(([label, value]) => `<section class="card"><div class="muted">${label}</div><div class="stat">${value}</div></section>`).join("");
  const reports = await API.reports("");
  const byWarehouse = {};
  reports.forEach((r) => { byWarehouse[r.warehouse] = (byWarehouse[r.warehouse] || 0) + Number(r.received || 0); });
  document.getElementById("chart").innerHTML = Object.entries(byWarehouse).map(([k, v]) => `<div><div class="muted">${escapeHtml(k)}</div><div class="progress-bar"><div class="progress-bar-fill" style="width:${Math.min(100, v)}%"></div></div><strong>${v}</strong></div>`).join("") || '<p class="muted">No receiving data yet.</p>';
})().catch((e) => toast(e.message, true));
