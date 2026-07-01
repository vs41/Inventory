package models

import "time"

type Role string

const (
	RoleCentralAdmin Role = "central_admin"
	RoleHubUser      Role = "hub_user"
)

type User struct {
	ID           string `json:"id"`
	Email        string `json:"email"`
	PasswordHash string `json:"-"`
	Role         Role   `json:"role"`
	Disabled     bool   `json:"disabled"`
}

type Warehouse struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	UserCount int    `json:"user_count,omitempty"`
}

type Invoice struct {
	InvoiceID   string `json:"invoice_id"`
	VendorName  string `json:"vendor_name"`
	WarehouseID string `json:"warehouse_id"`
	Status      string `json:"status"`
}

type InvoiceLine struct {
	InvoiceID        string `json:"invoice_id"`
	ItemSKU          string `json:"item_sku"`
	ItemName         string `json:"item_name"`
	ExpectedQuantity int    `json:"expected_quantity"`
	ReceivedQuantity int    `json:"received_quantity"`
	Variance         int    `json:"variance"`
}

type ScanEvent struct {
	InvoiceID   string    `json:"invoice_id"`
	ItemSKU     string    `json:"item_sku"`
	WarehouseID string    `json:"warehouse_id"`
	UserID      string    `json:"user_id"`
	ActionType  string    `json:"action_type"`
	Delta       int       `json:"delta"`
	Reason      string    `json:"reason"`
	Timestamp   time.Time `json:"timestamp"`
}
