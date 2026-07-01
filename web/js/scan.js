requireRole("hub_user");

const warehouseSelect = document.getElementById("warehouseSelect");
const invoiceSelect = document.getElementById("invoiceSelect");
const scanCard = document.getElementById("scanCard");
const scanInput = document.getElementById("scanInput");
const scanErr = document.getElementById("scanErr");
const progressCard = document.getElementById("progressCard");
const progressList = document.getElementById("progressList");
let currentInvoiceId = null;
let lastSku = "";
let eventSource = null;

async function init() {
  const warehouses = await API.myWarehouses();
  warehouseSelect.innerHTML = warehouses.map((w) => `<option value="${escapeHtml(w.id)}">${escapeHtml(w.name)} (${escapeHtml(w.id)})</option>`).join("");
  if (warehouses.length) await loadInvoices(warehouses[0].id);
}

async function loadInvoices(warehouseId) {
  const invoices = await API.invoicesForWarehouse(warehouseId);
  invoiceSelect.innerHTML = invoices.map((i) => `<option value="${escapeHtml(i.invoice_id)}">${escapeHtml(i.invoice_id)} - ${escapeHtml(i.vendor_name)} - ${escapeHtml(i.status)}</option>`).join("");
  if (invoices.length) selectInvoice(invoices[0].invoice_id);
}

function selectInvoice(invoiceId) {
  currentInvoiceId = invoiceId;
  scanCard.style.display = "block";
  progressCard.style.display = "block";
  scanInput.value = "";
  scanInput.focus();
  refreshProgress();
  if (eventSource) eventSource.close();
  eventSource = new EventSource(`/api/hub/progress/events?invoice_id=${encodeURIComponent(invoiceId)}&access_token=${encodeURIComponent(API.token())}`);
  eventSource.onmessage = (e) => renderProgress(JSON.parse(e.data));
}

warehouseSelect.addEventListener("change", (e) => loadInvoices(e.target.value));
invoiceSelect.addEventListener("change", (e) => selectInvoice(e.target.value));
document.addEventListener("click", () => scanInput.focus());

scanInput.addEventListener("keydown", async (e) => {
  if (e.key !== "Enter") return;
  const sku = scanInput.value.trim();
  scanInput.value = "";
  if (!sku || !currentInvoiceId) return;
  lastSku = sku;
  scanInput.style.borderColor = "var(--accent)";
  setTimeout(() => (scanInput.style.borderColor = ""), 180);
  const res = await API.scan(currentInvoiceId, sku);
  if (!res.ok) {
    const data = await res.json().catch(() => ({}));
    showError(data.error || `Scan rejected for SKU ${sku}`);
  }
  await refreshProgress();
});

document.getElementById("incBtn").onclick = () => adjust("inc");
document.getElementById("decBtn").onclick = () => adjust("dec");
document.getElementById("finishBtn").onclick = async () => {
  if (!currentInvoiceId) return;
  await API.finishReceiving(currentInvoiceId);
  toast("Receiving completed");
  await loadInvoices(warehouseSelect.value);
};

async function adjust(kind) {
  const sku = scanInput.value.trim() || lastSku;
  if (!sku) { showError("Scan or enter a SKU first"); return; }
  try {
    if (kind === "inc") await API.manualIncrement(currentInvoiceId, sku, "manual adjustment");
    else await API.manualDecrement(currentInvoiceId, sku, "manual adjustment");
    toast("Quantity updated");
    await refreshProgress();
  } catch (e) { showError(e.message); }
}

async function refreshProgress() {
  if (!currentInvoiceId) return;
  renderProgress(await API.progress(currentInvoiceId));
}

function renderProgress(lines) {
  progressList.innerHTML = lines.map((l) => {
    const pct = Math.min(100, Math.round((l.received_quantity / l.expected_quantity) * 100));
    const over = l.received_quantity > l.expected_quantity;
    return `<div class="line-item"><div><div class="name">${escapeHtml(l.item_name)}</div><div class="sku">${escapeHtml(l.item_sku)} - ${l.received_quantity}/${l.expected_quantity}</div></div><div class="progress-bar" style="width:160px;"><div class="progress-bar-fill ${over ? "over" : ""}" style="width:${pct}%;"></div></div></div>`;
  }).join("");
}

function showError(message) {
  scanErr.textContent = message;
  scanErr.style.display = "block";
  setTimeout(() => (scanErr.style.display = "none"), 3000);
}

init().catch((e) => showError(e.message));
