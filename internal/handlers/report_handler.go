package handlers

import (
	"encoding/csv"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/jackc/pgx/v5/pgxpool"
)

type ReportHandler struct {
	DB *pgxpool.Pool
}

func (h *ReportHandler) JSON(w http.ResponseWriter, r *http.Request) {
	rows, err := h.query(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	var out []map[string]interface{}
	for rows.Next() {
		var invoiceID, vendorName, whID, status, sku, name string
		var expected, received, variance int
		if err := rows.Scan(&invoiceID, &vendorName, &whID, &status, &sku, &name, &expected, &received, &variance); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		out = append(out, map[string]interface{}{"invoice": invoiceID, "vendor": vendorName, "warehouse": whID, "status": status, "sku": sku, "item": name, "expected": expected, "received": received, "variance": variance})
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(out)
}

func (h *ReportHandler) Reconciliation(w http.ResponseWriter, r *http.Request) {
	rows, err := h.query(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	format := r.URL.Query().Get("format")
	if format == "xlsx" {
		w.Header().Set("Content-Type", "application/vnd.ms-excel")
		w.Header().Set("Content-Disposition", "attachment; filename=reconciliation_report.xls")
	} else {
		w.Header().Set("Content-Type", "text/csv")
		w.Header().Set("Content-Disposition", "attachment; filename=reconciliation_report.csv")
	}
	cw := csv.NewWriter(w)
	_ = cw.Write([]string{"Invoice", "Vendor", "Warehouse", "Status", "SKU", "Item", "Expected", "Received", "Variance"})
	for rows.Next() {
		var invoiceID, vendorName, whID, status, sku, name string
		var expected, received, variance int
		if err := rows.Scan(&invoiceID, &vendorName, &whID, &status, &sku, &name, &expected, &received, &variance); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		_ = cw.Write([]string{invoiceID, vendorName, whID, status, sku, name, strconv.Itoa(expected), strconv.Itoa(received), strconv.Itoa(variance)})
	}
	cw.Flush()
}

func (h *ReportHandler) query(r *http.Request) (interface {
	Close()
	Next() bool
	Scan(...interface{}) error
}, error) {
	q := r.URL.Query()
	from, to := q.Get("from"), q.Get("to")
	warehouseID, vendor := q.Get("warehouse_id"), q.Get("vendor")
	status := q.Get("status")

	query := `
		SELECT i.invoice_id, i.vendor_name, i.warehouse_id, i.status, il.item_sku, il.item_name,
		       il.expected_quantity, COALESCE(rl.received_quantity, 0) AS received_quantity,
		       il.expected_quantity - COALESCE(rl.received_quantity, 0) AS variance
		FROM invoice_lines il
		JOIN invoices i ON i.invoice_id = il.invoice_id
		LEFT JOIN receiving_ledger rl ON rl.invoice_id = il.invoice_id AND rl.item_sku = il.item_sku
		WHERE ($1 = '' OR i.created_at >= $1::timestamptz)
		  AND ($2 = '' OR i.created_at <= $2::timestamptz)
		  AND ($3 = '' OR i.warehouse_id = $3)
		  AND ($4 = '' OR i.vendor_name = $4)
		  AND ($5 = '' OR i.status = $5)
		ORDER BY i.invoice_id, il.item_sku`

	return h.DB.Query(r.Context(), query, from, to, warehouseID, vendor, status)
}
