const API = {
  base: "",

  token() {
    return localStorage.getItem("ft_token");
  },

  async request(path, opts = {}) {
    const headers = opts.headers || {};
    if (this.token()) headers["Authorization"] = "Bearer " + this.token();
    if (opts.body && !(opts.body instanceof FormData)) {
      headers["Content-Type"] = "application/json";
    }
    const res = await fetch(this.base + path, { ...opts, headers });
    if (res.status === 401) {
      localStorage.removeItem("ft_token");
      window.location.href = "/index.html";
      return;
    }
    return res;
  },

  async json(path, opts = {}) {
    const res = await this.request(path, opts);
    const data = await res.json().catch(() => ({}));
    if (!res.ok) throw new Error(data.error || data.message || JSON.stringify(data) || "Request failed");
    return data;
  },

  async login(email, password) {
    const res = await fetch("/api/auth/login", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ email, password }),
    });
    if (!res.ok) throw new Error("Invalid credentials");
    const data = await res.json();
    localStorage.setItem("ft_token", data.token);
    localStorage.setItem("ft_role", data.role);
    return data;
  },

  logout() {
    this.request("/api/logout", { method: "POST" }).catch(() => {});
    localStorage.removeItem("ft_token");
    localStorage.removeItem("ft_role");
    window.location.href = "/index.html";
  },

  adminDashboard() { return this.json("/api/dashboard"); },
  hubDashboard() { return this.json("/api/hub-dashboard"); },
  users() { return this.json("/api/users"); },
  warehouses() { return this.json("/api/warehouses"); },
  createWarehouse(id, name) { return this.json("/api/warehouses", { method: "POST", body: JSON.stringify({ id, name }) }); },
  updateWarehouse(id, name) { return this.json(`/api/warehouses/${encodeURIComponent(id)}`, { method: "PUT", body: JSON.stringify({ name }) }); },
  deleteWarehouse(id) { return this.json(`/api/warehouses/${encodeURIComponent(id)}`, { method: "DELETE" }); },
  createUser(payload) { return this.json("/api/users", { method: "POST", body: JSON.stringify(payload) }); },
  updateUser(id, payload) { return this.json(`/api/users/${encodeURIComponent(id)}`, { method: "PUT", body: JSON.stringify(payload) }); },
  disableUser(id) { return this.json(`/api/users/${encodeURIComponent(id)}`, { method: "DELETE" }); },
  invoices() { return this.json("/api/invoices"); },
  audit() { return this.json("/api/audit"); },
  reports(params = "") { return this.json("/api/reports" + params); },
  history() { return this.json("/api/history"); },

  myWarehouses() {
    return this.request("/api/hub/warehouses").then((r) => r.json());
  },

  invoicesForWarehouse(warehouseId) {
    return this.request(`/api/hub/invoices?warehouse_id=${encodeURIComponent(warehouseId)}`).then((r) => r.json());
  },

  scan(invoiceId, itemSku) {
    return this.request("/api/hub/scan", {
      method: "POST",
      body: JSON.stringify({ invoice_id: invoiceId, item_sku: itemSku }),
    });
  },

  manualIncrement(invoiceId, itemSku, reason = "") {
    return this.json("/api/manual-increment", { method: "POST", body: JSON.stringify({ invoice_id: invoiceId, item_sku: itemSku, reason }) });
  },

  manualDecrement(invoiceId, itemSku, reason = "") {
    return this.json("/api/manual-decrement", { method: "POST", body: JSON.stringify({ invoice_id: invoiceId, item_sku: itemSku, reason }) });
  },

  finishReceiving(invoiceId) {
    return this.json("/api/finish-receiving", { method: "POST", body: JSON.stringify({ invoice_id: invoiceId }) });
  },

  progress(invoiceId) {
    return this.request(`/api/hub/progress?invoice_id=${encodeURIComponent(invoiceId)}`).then((r) => r.json());
  },

  uploadInvoice(file) {
    const form = new FormData();
    form.append("file", file);
    return this.request("/api/upload", { method: "POST", body: form });
  },
};
