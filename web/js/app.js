function requireRole(role) {
  if (!API.token()) window.location.href = "/index.html";
  const current = localStorage.getItem("ft_role");
  if (role && current !== role) window.location.href = current === "central_admin" ? "/admin/dashboard.html" : "/hub/dashboard.html";
}

function toast(message, danger = false) {
  let el = document.getElementById("toast");
  if (!el) {
    el = document.createElement("div");
    el.id = "toast";
    el.className = "toast";
    document.body.appendChild(el);
  }
  el.textContent = message;
  el.style.borderColor = danger ? "var(--danger)" : "var(--border)";
  el.style.display = "block";
  setTimeout(() => (el.style.display = "none"), 2800);
}

function setRows(tbody, rows, columns) {
  tbody.innerHTML = rows.map((row) => `<tr>${columns.map((c) => {
    const value = c.render ? c.render(row) : row[c.key] ?? "";
    return `<td>${c.raw ? value : escapeHtml(value)}</td>`;
  }).join("")}</tr>`).join("");
}

function escapeHtml(value) {
  return String(value).replace(/[&<>"']/g, (ch) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" }[ch]));
}

function wireSearch(input, getRows, render) {
  input.addEventListener("input", () => {
    const q = input.value.toLowerCase();
    render(getRows().filter((row) => JSON.stringify(row).toLowerCase().includes(q)));
  });
}

function downloadBlob(res, filename) {
  return res.blob().then((blob) => {
    const link = document.createElement("a");
    link.href = URL.createObjectURL(blob);
    link.download = filename;
    link.click();
    URL.revokeObjectURL(link.href);
  });
}
