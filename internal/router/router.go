package router

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"freshtrack/internal/config"
	"freshtrack/internal/handlers"
	"freshtrack/internal/middleware"
)

func New(cfg config.Config, db *pgxpool.Pool, rdb *redis.Client) http.Handler {
	r := chi.NewRouter()

	authH := &handlers.AuthHandler{DB: db, Cfg: cfg}
	adminH := &handlers.AdminHandler{DB: db}
	hubH := &handlers.HubHandler{DB: db, Redis: rdb}
	reportH := &handlers.ReportHandler{DB: db}

	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	r.Post("/api/auth/login", authH.Login)
	r.Post("/login", authH.Login)
	r.Post("/api/login", authH.Login)

	r.Group(func(r chi.Router) {
		r.Use(middleware.RequireAuth(cfg.JWTSecret))
		r.Post("/logout", authH.Logout)
		r.Post("/api/logout", authH.Logout)
		r.Get("/me", authH.Me)
		r.Get("/api/me", authH.Me)
	})

	r.Group(func(r chi.Router) {
		r.Use(middleware.RequireAuth(cfg.JWTSecret))
		r.Get("/api/invoices", hubH.AllInvoices)
		r.Get("/api/invoice/{id}", hubH.InvoiceDetail)
		r.Get("/api/progress", hubH.Progress)
		r.Get("/api/reports", reportH.JSON)
	})

	r.Group(func(r chi.Router) {
		r.Use(middleware.RequireAuth(cfg.JWTSecret))
		r.Use(middleware.RequireRole("central_admin"))
		r.Get("/api/dashboard", adminH.Dashboard)
		r.Get("/api/users", adminH.ListUsers)
		r.Post("/api/users", adminH.CreateUser)
		r.Put("/api/users/{id}", adminH.UpdateUser)
		r.Delete("/api/users/{id}", adminH.DisableUser)
		r.Get("/api/warehouses", adminH.ListWarehouses)
		r.Post("/api/warehouses", adminH.CreateWarehouse)
		r.Put("/api/warehouses/{id}", adminH.UpdateWarehouse)
		r.Delete("/api/warehouses/{id}", adminH.DeleteWarehouse)
		r.Post("/api/upload", adminH.UploadInvoice)
		r.Post("/api/override", hubH.Override)
		r.Get("/api/audit", adminH.Audit)
	})

	r.Group(func(r chi.Router) {
		r.Use(middleware.RequireAuth(cfg.JWTSecret))
		r.Use(middleware.RequireRole("hub_user"))
		r.Get("/api/hub-dashboard", hubH.Dashboard)
		r.Post("/api/scan", hubH.Scan)
		r.Post("/api/manual-increment", hubH.ManualIncrement)
		r.Post("/api/manual-decrement", hubH.ManualDecrement)
		r.Post("/api/finish-receiving", hubH.FinishReceiving)
		r.Get("/api/history", hubH.History)
	})

	r.Route("/api/admin", func(r chi.Router) {
		r.Use(middleware.RequireAuth(cfg.JWTSecret))
		r.Use(middleware.RequireRole("central_admin"))

		r.Get("/dashboard", adminH.Dashboard)
		r.Get("/warehouses", adminH.ListWarehouses)
		r.Post("/warehouses", adminH.CreateWarehouse)
		r.Put("/warehouses/{id}", adminH.UpdateWarehouse)
		r.Delete("/warehouses/{id}", adminH.DeleteWarehouse)
		r.Get("/users", adminH.ListUsers)
		r.Post("/users", adminH.CreateUser)
		r.Put("/users/{id}", adminH.UpdateUser)
		r.Delete("/users/{id}", adminH.DisableUser)
		r.Post("/users/map-warehouses", adminH.MapUserWarehouses)
		r.Post("/invoices/upload", adminH.UploadInvoice)
		r.Get("/audit", adminH.Audit)
		r.Get("/reports", reportH.JSON)
		r.Get("/reports/reconciliation", reportH.Reconciliation)
		r.Get("/users/{id}/warehouses", adminH.GetUserWarehouses)
		r.Put("/users/{id}/warehouses", adminH.UpdateUserWarehouses)
	})

	r.Route("/api/hub", func(r chi.Router) {
		r.Use(middleware.RequireAuth(cfg.JWTSecret))
		r.Use(middleware.RequireRole("hub_user"))

		r.Get("/dashboard", hubH.Dashboard)
		r.Get("/warehouses", hubH.MyWarehouses)
		r.Get("/invoices", hubH.InvoicesForWarehouse)
		r.Post("/scan", hubH.Scan)
		r.Post("/manual-increment", hubH.ManualIncrement)
		r.Post("/manual-decrement", hubH.ManualDecrement)
		r.Post("/finish-receiving", hubH.FinishReceiving)
		r.Get("/progress", hubH.Progress)
		r.Get("/progress/events", hubH.ProgressSSE)
		r.Get("/history", hubH.History)
	})

	fs := http.FileServer(http.Dir("./web"))
	r.Handle("/*", fs)

	return r
}
