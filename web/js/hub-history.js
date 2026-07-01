requireRole("hub_user");
(async () => {
  const rows = await API.history();
  setRows(document.getElementById("rows"), rows, [{ key: "timestamp" }, { key: "invoice" }, { key: "sku" }, { key: "warehouse" }, { key: "action" }, { key: "old_quantity" }, { key: "new_quantity" }, { key: "reason" }]);
})().catch((e) => toast(e.message, true));
