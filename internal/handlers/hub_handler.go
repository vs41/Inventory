package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"freshtrack/internal/middleware"
	"freshtrack/internal/models"
	"freshtrack/internal/redisclient"
)

type HubHandler struct {
	DB    *pgxpool.Pool
	Redis *redis.Client
}

type scanRequest struct {
	InvoiceID string `json:"invoice_id"`
	ItemSKU   string `json:"item_sku"`
	Reason    string `json:"reason"`
	Quantity  int    `json:"quantity"`
}

func (h *HubHandler) Dashboard(w http.ResponseWriter, r *http.Request) {
	userID, _ := r.Context().Value(middleware.CtxUserID).(string)
	row := h.DB.QueryRow(r.Context(), `
		SELECT
			(SELECT count(*) FROM user_warehouses WHERE user_id = $1),
			(SELECT count(*) FROM invoices i JOIN user_warehouses uw ON uw.warehouse_id = i.warehouse_id WHERE uw.user_id = $1 AND i.status IN ('Pending','Receiving')),
			(SELECT COALESCE(sum(new_quantity - old_quantity),0) FROM audit_log WHERE user_id = $1 AND action_type IN ('scan','manual_increment') AND ts::date = current_date),
			(SELECT count(*) FROM invoices i JOIN user_warehouses uw ON uw.warehouse_id = i.warehouse_id WHERE uw.user_id = $1 AND i.status = 'Completed' AND i.created_at::date = current_date)`, userID)
	var assigned, pendingInvoices, todayReceived, completedToday int
	if err := row.Scan(&assigned, &pendingInvoices, &todayReceived, &completedToday); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]int{"assigned_warehouses": assigned, "pending_invoices": pendingInvoices, "today_received": todayReceived, "completed_today": completedToday})
}

func (h *HubHandler) MyWarehouses(w http.ResponseWriter, r *http.Request) {
	userID, _ := r.Context().Value(middleware.CtxUserID).(string)
	rows, err := h.DB.Query(r.Context(), `SELECT wh.id, wh.name FROM warehouses wh JOIN user_warehouses uw ON uw.warehouse_id = wh.id WHERE uw.user_id = $1 ORDER BY wh.name`, userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	var out []models.Warehouse
	for rows.Next() {
		var wh models.Warehouse
		if err := rows.Scan(&wh.ID, &wh.Name); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		out = append(out, wh)
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *HubHandler) hasWarehouseAccess(r *http.Request, userID, warehouseID string) (bool, error) {
	var exists bool
	err := h.DB.QueryRow(r.Context(), `SELECT EXISTS(SELECT 1 FROM user_warehouses WHERE user_id = $1 AND warehouse_id = $2)`, userID, warehouseID).Scan(&exists)
	return exists, err
}

func (h *HubHandler) InvoicesForWarehouse(w http.ResponseWriter, r *http.Request) {
	userID, _ := r.Context().Value(middleware.CtxUserID).(string)
	whID := r.URL.Query().Get("warehouse_id")
	ok, err := h.hasWarehouseAccess(r, userID, whID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if !ok {
		http.Error(w, "forbidden: not mapped to this warehouse", http.StatusForbidden)
		return
	}
	rows, err := h.DB.Query(r.Context(), `SELECT invoice_id, vendor_name, warehouse_id, status FROM invoices WHERE warehouse_id = $1 AND status IN ('Pending','Receiving') ORDER BY created_at DESC`, whID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	var out []models.Invoice
	for rows.Next() {
		var inv models.Invoice
		if err := rows.Scan(&inv.InvoiceID, &inv.VendorName, &inv.WarehouseID, &inv.Status); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		out = append(out, inv)
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *HubHandler) AllInvoices(w http.ResponseWriter, r *http.Request) {
	userID, _ := r.Context().Value(middleware.CtxUserID).(string)
	role, _ := r.Context().Value(middleware.CtxRole).(string)
	var rows pgx.Rows
	var err error
	if role == "central_admin" {
		rows, err = h.DB.Query(r.Context(), `SELECT invoice_id, vendor_name, warehouse_id, status FROM invoices ORDER BY created_at DESC`)
	} else {
		rows, err = h.DB.Query(r.Context(), `SELECT i.invoice_id, i.vendor_name, i.warehouse_id, i.status FROM invoices i JOIN user_warehouses uw ON uw.warehouse_id = i.warehouse_id WHERE uw.user_id = $1 ORDER BY i.created_at DESC`, userID)
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	var out []models.Invoice
	for rows.Next() {
		var inv models.Invoice
		if err := rows.Scan(&inv.InvoiceID, &inv.VendorName, &inv.WarehouseID, &inv.Status); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		out = append(out, inv)
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *HubHandler) InvoiceDetail(w http.ResponseWriter, r *http.Request) {
	invoiceID := chi.URLParam(r, "id")
	h.writeProgress(w, r, invoiceID)
}

func (h *HubHandler) Scan(w http.ResponseWriter, r *http.Request) {
	userID, _ := r.Context().Value(middleware.CtxUserID).(string)
	var req scanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if req.InvoiceID == "" || req.ItemSKU == "" {
		http.Error(w, "invoice_id and item_sku are required", http.StatusBadRequest)
		return
	}
	whID, ok := h.authorizeInvoice(w, r, userID, req.InvoiceID)
	if !ok {
		return
	}
	var skuExists bool
	if err := h.DB.QueryRow(r.Context(), `SELECT EXISTS(SELECT 1 FROM invoice_lines WHERE invoice_id = $1 AND item_sku = $2)`, req.InvoiceID, req.ItemSKU).Scan(&skuExists); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if !skuExists {
		http.Error(w, "sku not on this invoice", http.StatusUnprocessableEntity)
		return
	}
	dupKey := fmt.Sprintf("freshtrack:dup:%s:%s:%s", req.InvoiceID, req.ItemSKU, userID)
	accepted, err := h.Redis.SetNX(r.Context(), dupKey, "1", 700*time.Millisecond).Result()
	if err != nil {
		http.Error(w, "duplicate protection unavailable: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if !accepted {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "duplicate scan ignored"})
		return
	}
	event := models.ScanEvent{InvoiceID: req.InvoiceID, ItemSKU: req.ItemSKU, WarehouseID: whID, UserID: userID, ActionType: "scan", Delta: 1, Timestamp: time.Now()}
	payload, _ := json.Marshal(event)
	if err := h.Redis.XAdd(r.Context(), &redis.XAddArgs{Stream: redisclient.ScanStreamKey, Values: map[string]interface{}{"payload": string(payload)}}).Err(); err != nil {
		http.Error(w, "queue error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]string{"status": "queued"})
}

func (h *HubHandler) ManualIncrement(w http.ResponseWriter, r *http.Request) {
	h.manualAdjust(w, r, "manual_increment", 1)
}

func (h *HubHandler) ManualDecrement(w http.ResponseWriter, r *http.Request) {
	h.manualAdjust(w, r, "manual_decrement", -1)
}

func (h *HubHandler) Override(w http.ResponseWriter, r *http.Request) {
	userID, _ := r.Context().Value(middleware.CtxUserID).(string)
	var req scanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if req.Reason == "" {
		http.Error(w, "reason is required", http.StatusBadRequest)
		return
	}
	whID, ok := h.invoiceWarehouse(w, r, req.InvoiceID)
	if !ok {
		return
	}
	oldQty, newQty, err := h.applyQuantity(r, req.InvoiceID, req.ItemSKU, whID, userID, "override", req.Quantity, req.Reason, true)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}
	writeJSON(w, http.StatusOK, map[string]int{"old_quantity": oldQty, "new_quantity": newQty})
}

func (h *HubHandler) manualAdjust(w http.ResponseWriter, r *http.Request, action string, delta int) {
	userID, _ := r.Context().Value(middleware.CtxUserID).(string)
	var req scanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	whID, ok := h.authorizeInvoice(w, r, userID, req.InvoiceID)
	if !ok {
		return
	}
	oldQty, newQty, err := h.applyQuantity(r, req.InvoiceID, req.ItemSKU, whID, userID, action, delta, req.Reason, false)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}
	writeJSON(w, http.StatusOK, map[string]int{"old_quantity": oldQty, "new_quantity": newQty})
}

func (h *HubHandler) applyQuantity(r *http.Request, invoiceID, sku, whID, userID, action string, value int, reason string, absolute bool) (int, int, error) {
	tx, err := h.DB.Begin(r.Context())
	if err != nil {
		return 0, 0, err
	}
	defer tx.Rollback(r.Context())
	var oldQty int
	if err := tx.QueryRow(r.Context(), `SELECT received_quantity FROM receiving_ledger WHERE invoice_id = $1 AND item_sku = $2 FOR UPDATE`, invoiceID, sku).Scan(&oldQty); err != nil {
		return 0, 0, fmt.Errorf("invoice line not found")
	}
	newQty := oldQty + value
	if absolute {
		newQty = value
	}
	if newQty < 0 {
		return oldQty, oldQty, fmt.Errorf("received quantity cannot be negative")
	}
	if _, err := tx.Exec(r.Context(), `UPDATE receiving_ledger SET received_quantity = $1, last_updated = now() WHERE invoice_id = $2 AND item_sku = $3`, newQty, invoiceID, sku); err != nil {
		return 0, 0, err
	}
	if _, err := tx.Exec(r.Context(), `INSERT INTO audit_log (invoice_id, item_sku, warehouse_id, user_id, action_type, old_quantity, new_quantity, delta, reason) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`, invoiceID, sku, whID, userID, action, oldQty, newQty, newQty-oldQty, reason); err != nil {
		return 0, 0, err
	}
	if err := updateInvoiceStatus(r.Context(), tx, invoiceID); err != nil {
		return 0, 0, err
	}
	return oldQty, newQty, tx.Commit(r.Context())
}

func (h *HubHandler) FinishReceiving(w http.ResponseWriter, r *http.Request) {
	userID, _ := r.Context().Value(middleware.CtxUserID).(string)
	var req scanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if _, ok := h.authorizeInvoice(w, r, userID, req.InvoiceID); !ok {
		return
	}
	if _, err := h.DB.Exec(r.Context(), `UPDATE invoices SET status = 'Completed' WHERE invoice_id = $1`, req.InvoiceID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "Completed"})
}

func (h *HubHandler) Progress(w http.ResponseWriter, r *http.Request) {
	h.writeProgress(w, r, r.URL.Query().Get("invoice_id"))
}

func (h *HubHandler) writeProgress(w http.ResponseWriter, r *http.Request, invoiceID string) {
	rows, err := h.DB.Query(r.Context(), `
		SELECT il.invoice_id, il.item_sku, il.item_name, il.expected_quantity, COALESCE(rl.received_quantity, 0)
		FROM invoice_lines il
		LEFT JOIN receiving_ledger rl ON rl.invoice_id = il.invoice_id AND rl.item_sku = il.item_sku
		WHERE il.invoice_id = $1
		ORDER BY il.item_sku`, invoiceID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	var out []models.InvoiceLine
	for rows.Next() {
		var l models.InvoiceLine
		if err := rows.Scan(&l.InvoiceID, &l.ItemSKU, &l.ItemName, &l.ExpectedQuantity, &l.ReceivedQuantity); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		l.Variance = l.ExpectedQuantity - l.ReceivedQuantity
		out = append(out, l)
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *HubHandler) ProgressSSE(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	invoiceID := r.URL.Query().Get("invoice_id")
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
			rows, err := h.progressRows(r, invoiceID)
			if err != nil {
				return
			}
			b, _ := json.Marshal(rows)
			fmt.Fprintf(w, "data: %s\n\n", b)
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
		}
	}
}

func (h *HubHandler) progressRows(r *http.Request, invoiceID string) ([]models.InvoiceLine, error) {
	rows, err := h.DB.Query(r.Context(), `SELECT il.invoice_id, il.item_sku, il.item_name, il.expected_quantity, COALESCE(rl.received_quantity, 0) FROM invoice_lines il LEFT JOIN receiving_ledger rl ON rl.invoice_id = il.invoice_id AND rl.item_sku = il.item_sku WHERE il.invoice_id = $1`, invoiceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.InvoiceLine
	for rows.Next() {
		var l models.InvoiceLine
		if err := rows.Scan(&l.InvoiceID, &l.ItemSKU, &l.ItemName, &l.ExpectedQuantity, &l.ReceivedQuantity); err != nil {
			return nil, err
		}
		l.Variance = l.ExpectedQuantity - l.ReceivedQuantity
		out = append(out, l)
	}
	return out, nil
}

func (h *HubHandler) History(w http.ResponseWriter, r *http.Request) {
	userID, _ := r.Context().Value(middleware.CtxUserID).(string)
	rows, err := h.DB.Query(r.Context(), `SELECT ts, invoice_id, item_sku, warehouse_id, action_type, old_quantity, new_quantity, reason FROM audit_log WHERE user_id = $1 ORDER BY ts DESC LIMIT 200`, userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	var out []map[string]interface{}
	for rows.Next() {
		var ts time.Time
		var invoiceID, sku, whID, action, reason string
		var oldQty, newQty int
		if err := rows.Scan(&ts, &invoiceID, &sku, &whID, &action, &oldQty, &newQty, &reason); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		out = append(out, map[string]interface{}{"timestamp": ts.Format(time.RFC3339), "invoice": invoiceID, "sku": sku, "warehouse": whID, "action": action, "old_quantity": oldQty, "new_quantity": newQty, "reason": reason})
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *HubHandler) authorizeInvoice(w http.ResponseWriter, r *http.Request, userID, invoiceID string) (string, bool) {
	whID, ok := h.invoiceWarehouse(w, r, invoiceID)
	if !ok {
		return "", false
	}
	okAccess, err := h.hasWarehouseAccess(r, userID, whID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return "", false
	}
	if !okAccess {
		http.Error(w, "forbidden: not mapped to this warehouse", http.StatusForbidden)
		return "", false
	}
	return whID, true
}

func (h *HubHandler) invoiceWarehouse(w http.ResponseWriter, r *http.Request, invoiceID string) (string, bool) {
	var whID string
	if err := h.DB.QueryRow(r.Context(), `SELECT warehouse_id FROM invoices WHERE invoice_id = $1`, invoiceID).Scan(&whID); err != nil {
		http.Error(w, "invoice not found", http.StatusNotFound)
		return "", false
	}
	return whID, true
}

func updateInvoiceStatus(ctx context.Context, tx pgx.Tx, invoiceID string) error {
	var received, expected int
	if err := tx.QueryRow(ctx, `
		SELECT COALESCE(sum(rl.received_quantity),0), COALESCE(sum(il.expected_quantity),0)
		FROM invoice_lines il
		LEFT JOIN receiving_ledger rl ON rl.invoice_id = il.invoice_id AND rl.item_sku = il.item_sku
		WHERE il.invoice_id = $1`, invoiceID).Scan(&received, &expected); err != nil {
		return err
	}
	status := "Pending"
	if received > 0 {
		status = "Receiving"
	}
	if expected > 0 && received >= expected {
		status = "Completed"
	}
	_, err := tx.Exec(ctx, `UPDATE invoices SET status = $1 WHERE invoice_id = $2`, status, invoiceID)
	return err
}
