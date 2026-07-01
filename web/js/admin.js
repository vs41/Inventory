if (!API.token()) window.location.href = "/index.html";

document.getElementById("uploadBtn").addEventListener("click", async () => {
  const file = document.getElementById("invoiceFile").files[0];
  const resultEl = document.getElementById("uploadResult");
  if (!file) { resultEl.textContent = "Choose a CSV file first."; return; }
  const res = await API.uploadInvoice(file);
  const data = await res.json();
  if (res.ok) {
    resultEl.style.color = "var(--accent)";
    resultEl.textContent = `Ingested ${data.lines} lines across ${data.invoices} invoice(s).`;
  } else {
    resultEl.style.color = "var(--danger)";
    resultEl.textContent = (data.errors || [data.error || "Upload failed"]).join("; ");
  }
});

document.getElementById("addWhBtn").addEventListener("click", async () => {
  const id = document.getElementById("whId").value.trim();
  const name = document.getElementById("whName").value.trim();
  if (!id || !name) return;
  const res = await API.request("/api/admin/warehouses", {
    method: "POST",
    body: JSON.stringify({ id, name }),
  });
  alert(res.ok ? "Warehouse created." : "Failed to create warehouse.");
});

document.getElementById("createUserBtn").addEventListener("click", async () => {
  const email = document.getElementById("newUserEmail").value.trim();
  const password = document.getElementById("newUserPassword").value;
  const role = document.getElementById("newUserRole").value;
  const resultEl = document.getElementById("createUserResult");
  if (!email || !password) { resultEl.textContent = "Email and password required."; return; }

  const res = await API.request("/api/admin/users", {
    method: "POST",
    body: JSON.stringify({ email, password, role }),
  });
  const data = await res.json();
  if (res.ok) {
    resultEl.style.color = "var(--accent)";
    resultEl.textContent = `Created ${data.email} (${data.role}). ID: ${data.id}`;
    document.getElementById("mapUserId").value = data.id; // convenience: pre-fill mapping field
    document.getElementById("newUserEmail").value = "";
    document.getElementById("newUserPassword").value = "";
  } else {
    resultEl.style.color = "var(--danger)";
    resultEl.textContent = data.error || "Failed to create user (email may already exist).";
  }
});

document.getElementById("mapBtn").addEventListener("click", async () => {
  const userId = document.getElementById("mapUserId").value.trim();
  const whIds = document.getElementById("mapWhIds").value.split(",").map(s => s.trim()).filter(Boolean);
  if (!userId || whIds.length === 0) return;
  const res = await API.request("/api/admin/users/map-warehouses", {
    method: "POST",
    body: JSON.stringify({ user_id: userId, warehouse_ids: whIds }),
  });
  alert(res.ok ? "Mapped." : "Failed to map user.");
});

document.getElementById("repBtn").addEventListener("click", () => {
  const wh = document.getElementById("repWh").value.trim();
  const vendor = document.getElementById("repVendor").value.trim();
  const params = new URLSearchParams();
  if (wh) params.set("warehouse_id", wh);
  if (vendor) params.set("vendor", vendor);
  const url = `/api/admin/reports/reconciliation?${params.toString()}`;
  API.request(url).then(async (res) => {
    const blob = await res.blob();
    const link = document.createElement("a");
    link.href = URL.createObjectURL(blob);
    link.download = "reconciliation_report.csv";
    link.click();
  });
});
