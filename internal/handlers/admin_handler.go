package handlers

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"freshtrack/internal/auth"
)

type AdminHandler struct {
	DB *pgxpool.Pool
}

type userRequest struct {
	Email        string   `json:"email"`
	Password     string   `json:"password"`
	Role         string   `json:"role"`
	Disabled     bool     `json:"disabled"`
	WarehouseIDs []string `json:"warehouse_ids"`
}

type warehouseRequest struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func (h *AdminHandler) Dashboard(w http.ResponseWriter, r *http.Request) {
	row := h.DB.QueryRow(r.Context(), `
		SELECT
			(SELECT count(*) FROM warehouses),
			(SELECT count(*) FROM users WHERE disabled = false),
			(SELECT count(*) FROM invoices WHERE status = 'Pending'),
			(SELECT count(*) FROM invoices WHERE status = 'Receiving'),
			(SELECT count(*) FROM invoices WHERE status = 'Completed')`)
	var totalWarehouses, totalUsers, pending, receiving, completed int
	if err := row.Scan(&totalWarehouses, &totalUsers, &pending, &receiving, &completed); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]int{
		"total_warehouses": totalWarehouses,
		"total_users":      totalUsers,
		"pending":          pending,
		"receiving":        receiving,
		"completed":        completed,
	})
}

func (h *AdminHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	rows, err := h.DB.Query(r.Context(), `
		SELECT u.id, u.email, u.role, u.disabled, COALESCE(string_agg(uw.warehouse_id, ',' ORDER BY uw.warehouse_id), '')
		FROM users u
		LEFT JOIN user_warehouses uw ON uw.user_id = u.id
		GROUP BY u.id
		ORDER BY u.email`)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	var out []map[string]interface{}
	for rows.Next() {
		var id, email, role, whCSV string
		var disabled bool
		if err := rows.Scan(&id, &email, &role, &disabled, &whCSV); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		warehouses := []string{}
		if whCSV != "" {
			warehouses = strings.Split(whCSV, ",")
		}
		out = append(out, map[string]interface{}{"id": id, "email": email, "role": role, "disabled": disabled, "warehouse_ids": warehouses})
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *AdminHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
	var req userRequest
	//log.Panicln()
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if err := validateUserRequest(req, true); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		http.Error(w, "could not hash password", http.StatusInternalServerError)
		return
	}
	tx, err := h.DB.Begin(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer tx.Rollback(r.Context())
	var id string
	if err := tx.QueryRow(r.Context(), `INSERT INTO users (email, password_hash, role, disabled) VALUES ($1, $2, $3, $4) RETURNING id`, req.Email, hash, req.Role, req.Disabled).Scan(&id); err != nil {
		http.Error(w, "could not create user: "+err.Error(), http.StatusConflict)
		return
	}
	if err := replaceWarehouseMappings(r.Context(), tx, id, req.WarehouseIDs); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]interface{}{"id": id, "email": req.Email, "role": req.Role, "disabled": req.Disabled, "warehouse_ids": req.WarehouseIDs})
}

func (h *AdminHandler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req userRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if err := validateUserRequest(req, false); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	tx, err := h.DB.Begin(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer tx.Rollback(r.Context())
	if req.Password != "" {
		hash, err := auth.HashPassword(req.Password)
		if err != nil {
			http.Error(w, "could not hash password", http.StatusInternalServerError)
			return
		}
		_, err = tx.Exec(r.Context(), `UPDATE users SET email = $1, role = $2, disabled = $3, password_hash = $4 WHERE id = $5`, req.Email, req.Role, req.Disabled, hash, id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}
	} else if _, err := tx.Exec(r.Context(), `UPDATE users SET email = $1, role = $2, disabled = $3 WHERE id = $4`, req.Email, req.Role, req.Disabled, id); err != nil {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}
	if err := replaceWarehouseMappings(r.Context(), tx, id, req.WarehouseIDs); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

func (h *AdminHandler) DisableUser(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if _, err := h.DB.Exec(r.Context(), `UPDATE users SET disabled = true WHERE id = $1`, id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "disabled"})
}

func validateUserRequest(req userRequest, requirePassword bool) error {
	if strings.TrimSpace(req.Email) == "" {
		return fmt.Errorf("email is required")
	}
	if requirePassword && len(req.Password) < 8 {
		return fmt.Errorf("password must be at least 8 characters")
	}
	if req.Password != "" && len(req.Password) < 8 {
		return fmt.Errorf("password must be at least 8 characters")
	}
	if req.Role != "central_admin" && req.Role != "hub_user" {
		return fmt.Errorf("role must be central_admin or hub_user")
	}
	return nil
}

func replaceWarehouseMappings(ctx context.Context, tx pgx.Tx, userID string, warehouseIDs []string) error {
	if _, err := tx.Exec(ctx, `DELETE FROM user_warehouses WHERE user_id = $1`, userID); err != nil {
		return err
	}
	for _, whID := range warehouseIDs {
		whID = strings.TrimSpace(whID)
		if whID == "" {
			continue
		}
		if _, err := tx.Exec(ctx, `INSERT INTO user_warehouses (user_id, warehouse_id) VALUES ($1, $2)`, userID, whID); err != nil {
			return fmt.Errorf("warehouse %s cannot be assigned: %w", whID, err)
		}
	}
	return nil
}

func (h *AdminHandler) ListWarehouses(w http.ResponseWriter, r *http.Request) {
	rows, err := h.DB.Query(r.Context(), `
		SELECT w.id, w.name, count(uw.user_id)
		FROM warehouses w
		LEFT JOIN user_warehouses uw ON uw.warehouse_id = w.id
		GROUP BY w.id, w.name
		ORDER BY w.id`)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	var out []map[string]interface{}
	for rows.Next() {
		var id, name string
		var users int
		if err := rows.Scan(&id, &name, &users); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		out = append(out, map[string]interface{}{"id": id, "name": name, "user_count": users})
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *AdminHandler) CreateWarehouse(w http.ResponseWriter, r *http.Request) {
	var req warehouseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(req.ID) == "" || strings.TrimSpace(req.Name) == "" {
		http.Error(w, "id and name are required", http.StatusBadRequest)
		return
	}
	if _, err := h.DB.Exec(r.Context(), `INSERT INTO warehouses (id, name) VALUES ($1, $2)`, req.ID, req.Name); err != nil {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}
	writeJSON(w, http.StatusCreated, req)
}

func (h *AdminHandler) UpdateWarehouse(w http.ResponseWriter, r *http.Request) {
	var req warehouseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(req.Name) == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}
	if _, err := h.DB.Exec(r.Context(), `UPDATE warehouses SET name = $1 WHERE id = $2`, req.Name, chi.URLParam(r, "id")); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

func (h *AdminHandler) DeleteWarehouse(w http.ResponseWriter, r *http.Request) {
	if _, err := h.DB.Exec(r.Context(), `DELETE FROM warehouses WHERE id = $1`, chi.URLParam(r, "id")); err != nil {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

type mapUserWarehouseRequest struct {
	UserID       string   `json:"user_id"`
	WarehouseIDs []string `json:"warehouse_ids"`
}

func (h *AdminHandler) MapUserWarehouses(w http.ResponseWriter, r *http.Request) {
	var req mapUserWarehouseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	tx, err := h.DB.Begin(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer tx.Rollback(r.Context())
	if err := replaceWarehouseMappings(r.Context(), tx, req.UserID, req.WarehouseIDs); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "mapped"})
}

type invoiceUploadLine struct {
	InvoiceID, Vendor, WarehouseID, SKU, Name string
	Qty                                       int
}

func (h *AdminHandler) UploadInvoice(w http.ResponseWriter, r *http.Request) {
	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "missing file field 'file'", http.StatusBadRequest)
		return
	}
	defer file.Close()
	var rows [][]string
	if strings.EqualFold(filepath.Ext(header.Filename), ".xlsx") {
		rows, err = readXLSXRows(file)
	} else {
		rows, err = csv.NewReader(file).ReadAll()
	}
	if err != nil {
		http.Error(w, "invalid invoice file: "+err.Error(), http.StatusBadRequest)
		return
	}
	lines, errs := h.validateInvoiceRows(r.Context(), rows)
	if len(errs) > 0 {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]interface{}{"status": "rejected", "errors": errs})
		return
	}
	if err := h.insertInvoiceLines(r.Context(), lines); err != nil {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}
	invoices := map[string]bool{}
	for _, l := range lines {
		invoices[l.InvoiceID] = true
	}
	writeJSON(w, http.StatusCreated, map[string]interface{}{"status": "ingested", "lines": len(lines), "invoices": len(invoices)})
}

func (h *AdminHandler) validateInvoiceRows(ctx context.Context, rows [][]string) ([]invoiceUploadLine, []string) {
	if len(rows) < 2 {
		return nil, []string{"row 1: file has no data rows"}
	}
	var lines []invoiceUploadLine
	var errs []string
	seenInvoices := map[string]string{}
	for i, row := range rows[1:] {
		rowNum := i + 2
		rowErr := false
		if len(row) < 6 {
			errs = append(errs, fmt.Sprintf("row %d: expected 6 columns", rowNum))
			rowErr = true
			continue
		}
		for j := range row {
			row[j] = strings.TrimSpace(row[j])
		}
		qty, err := strconv.Atoi(row[5])
		if err != nil || qty <= 0 {
			errs = append(errs, fmt.Sprintf("row %d: quantity must be greater than 0", rowNum))
			rowErr = true
		}
		if row[0] == "" {
			errs = append(errs, fmt.Sprintf("row %d: invoice id is required", rowNum))
			rowErr = true
		}
		if row[3] == "" {
			errs = append(errs, fmt.Sprintf("row %d: sku is required", rowNum))
			rowErr = true
		}
		if existingWarehouse, ok := seenInvoices[row[0]]; ok && existingWarehouse != row[2] {
			errs = append(errs, fmt.Sprintf("row %d: invoice %s appears with multiple warehouses", rowNum, row[0]))
			rowErr = true
		}
		seenInvoices[row[0]] = row[2]
		var whExists bool
		if err := h.DB.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM warehouses WHERE id = $1)`, row[2]).Scan(&whExists); err != nil {
			errs = append(errs, fmt.Sprintf("row %d: warehouse lookup failed", rowNum))
			rowErr = true
		} else if !whExists {
			errs = append(errs, fmt.Sprintf("row %d: warehouse %q does not exist", rowNum, row[2]))
			rowErr = true
		}
		var invoiceExists bool
		if err := h.DB.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM invoices WHERE invoice_id = $1)`, row[0]).Scan(&invoiceExists); err != nil {
			errs = append(errs, fmt.Sprintf("row %d: invoice lookup failed", rowNum))
			rowErr = true
		} else if invoiceExists {
			errs = append(errs, fmt.Sprintf("row %d: invoice %q already exists", rowNum, row[0]))
			rowErr = true
		}
		if !rowErr {
			lines = append(lines, invoiceUploadLine{InvoiceID: row[0], Vendor: row[1], WarehouseID: row[2], SKU: row[3], Name: row[4], Qty: qty})
		}
	}
	return lines, errs
}

func (h *AdminHandler) insertInvoiceLines(ctx context.Context, lines []invoiceUploadLine) error {
	tx, err := h.DB.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	seenInvoices := map[string]bool{}
	for _, l := range lines {
		if !seenInvoices[l.InvoiceID] {
			if _, err := tx.Exec(ctx, `INSERT INTO invoices (invoice_id, vendor_name, warehouse_id, status) VALUES ($1, $2, $3, 'Pending')`, l.InvoiceID, l.Vendor, l.WarehouseID); err != nil {
				return err
			}
			seenInvoices[l.InvoiceID] = true
		}
		if _, err := tx.Exec(ctx, `INSERT INTO invoice_lines (invoice_id, item_sku, item_name, expected_quantity) VALUES ($1, $2, $3, $4)`, l.InvoiceID, l.SKU, l.Name, l.Qty); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `INSERT INTO receiving_ledger (invoice_id, item_sku, received_quantity) VALUES ($1, $2, 0)`, l.InvoiceID, l.SKU); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (h *AdminHandler) Audit(w http.ResponseWriter, r *http.Request) {
	rows, err := h.DB.Query(r.Context(), `
		SELECT a.id, a.ts, a.invoice_id, a.item_sku, a.warehouse_id, u.email, a.action_type, a.old_quantity, a.new_quantity, a.reason
		FROM audit_log a
		JOIN users u ON u.id = a.user_id
		ORDER BY a.ts DESC
		LIMIT 500`)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	var out []map[string]interface{}
	for rows.Next() {
		var id, oldQty, newQty int
		var ts time.Time
		var invoiceID, sku, whID, email, action, reason string
		if err := rows.Scan(&id, &ts, &invoiceID, &sku, &whID, &email, &action, &oldQty, &newQty, &reason); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		out = append(out, map[string]interface{}{"id": id, "timestamp": ts.Format(time.RFC3339), "invoice": invoiceID, "sku": sku, "warehouse": whID, "user": email, "action": action, "old_quantity": oldQty, "new_quantity": newQty, "reason": reason})
	}
	writeJSON(w, http.StatusOK, out)
}

func readXLSXRows(r io.Reader) ([][]string, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, err
	}
	shared := []string{}
	for _, f := range zr.File {
		if f.Name == "xl/sharedStrings.xml" {
			rc, err := f.Open()
			if err != nil {
				return nil, err
			}
			shared, err = parseSharedStrings(rc)
			rc.Close()
			if err != nil {
				return nil, err
			}
		}
	}
	for _, f := range zr.File {
		if f.Name == "xl/worksheets/sheet1.xml" {
			rc, err := f.Open()
			if err != nil {
				return nil, err
			}
			defer rc.Close()
			return parseSheet(rc, shared)
		}
	}
	return nil, fmt.Errorf("sheet1.xml not found")
}

func parseSharedStrings(r io.Reader) ([]string, error) {
	type text struct {
		Value string `xml:",chardata"`
	}
	type item struct {
		Texts []text `xml:"t"`
	}
	type sst struct {
		Items []item `xml:"si"`
	}
	var doc sst
	if err := xml.NewDecoder(r).Decode(&doc); err != nil {
		return nil, err
	}
	out := make([]string, 0, len(doc.Items))
	for _, it := range doc.Items {
		var b strings.Builder
		for _, t := range it.Texts {
			b.WriteString(t.Value)
		}
		out = append(out, b.String())
	}
	return out, nil
}

func parseSheet(r io.Reader, shared []string) ([][]string, error) {
	type cell struct {
		Ref   string `xml:"r,attr"`
		Type  string `xml:"t,attr"`
		Value string `xml:"v"`
	}
	type row struct {
		Cells []cell `xml:"c"`
	}
	type sheet struct {
		Rows []row `xml:"sheetData>row"`
	}
	var doc sheet
	if err := xml.NewDecoder(r).Decode(&doc); err != nil {
		return nil, err
	}
	var rows [][]string
	for _, sr := range doc.Rows {
		out := make([]string, 6)
		for _, c := range sr.Cells {
			idx := columnIndex(c.Ref)
			if idx < 0 || idx >= len(out) {
				continue
			}
			val := c.Value
			if c.Type == "s" {
				n, _ := strconv.Atoi(c.Value)
				if n >= 0 && n < len(shared) {
					val = shared[n]
				}
			}
			out[idx] = val
		}
		rows = append(rows, out)
	}
	return rows, nil
}

func columnIndex(ref string) int {
	if ref == "" {
		return -1
	}
	col := 0
	for _, ch := range ref {
		if ch < 'A' || ch > 'Z' {
			break
		}
		col = col*26 + int(ch-'A'+1)
	}
	return col - 1
}

func (h *AdminHandler) GetUserWarehouses(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "id")

	rows, err := h.DB.Query(r.Context(), `
		SELECT warehouse_id
		FROM user_warehouses
		WHERE user_id = $1
		ORDER BY warehouse_id
	`, userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var warehouseIDs []string

	for rows.Next() {
		var id string

		if err := rows.Scan(&id); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		warehouseIDs = append(warehouseIDs, id)
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"user_id":       userID,
		"warehouse_ids": warehouseIDs,
	})
}

func (h *AdminHandler) UpdateUserWarehouses(w http.ResponseWriter, r *http.Request) {

	userID := chi.URLParam(r, "id")

	var req struct {
		WarehouseIDs []string `json:"warehouse_ids"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	tx, err := h.DB.Begin(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer tx.Rollback(r.Context())

	if err := replaceWarehouseMappings(
		r.Context(),
		tx,
		userID,
		req.WarehouseIDs,
	); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := tx.Commit(r.Context()); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"status": "updated",
	})
}
