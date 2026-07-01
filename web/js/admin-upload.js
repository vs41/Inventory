requireRole("central_admin");
document.getElementById("uploadBtn").onclick = async () => {
  const file = document.getElementById("invoiceFile").files[0];
  const result = document.getElementById("result");
  if (!file) { toast("Choose a file", true); return; }
  result.innerHTML = '<div class="spinner"></div>';
  const res = await API.uploadInvoice(file);
  const data = await res.json().catch(() => ({}));
  if (res.ok) {
    result.textContent = `Ingested ${data.lines} lines across ${data.invoices} invoice(s).`;
    toast("Invoice uploaded");
  } else {
    result.textContent = (data.errors || [data.error || "Upload failed"]).join("\n");
    toast("Upload rejected", true);
  }
};
