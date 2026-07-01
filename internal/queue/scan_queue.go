package queue

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"freshtrack/internal/models"
	"freshtrack/internal/redisclient"
)

const consumerGroup = "freshtrack-workers"
const consumerName = "worker-1"

func StartScanWorker(ctx context.Context, rdb *redis.Client, db *pgxpool.Pool) {

	_ = rdb.XGroupCreateMkStream(ctx, redisclient.ScanStreamKey, consumerGroup, "0").Err()

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		res, err := rdb.XReadGroup(ctx, &redis.XReadGroupArgs{
			Group:    consumerGroup,
			Consumer: consumerName,
			Streams:  []string{redisclient.ScanStreamKey, ">"},
			Count:    50,
			Block:    2 * time.Second,
		}).Result()
		if err != nil {
			if err != redis.Nil {
				log.Printf("scan worker read error: %v", err)
				time.Sleep(500 * time.Millisecond)
			}
			continue
		}

		for _, stream := range res {
			for _, msg := range stream.Messages {
				raw, _ := msg.Values["payload"].(string)
				var event models.ScanEvent
				if err := json.Unmarshal([]byte(raw), &event); err != nil {
					log.Printf("bad scan event payload: %v", err)
					rdb.XAck(ctx, redisclient.ScanStreamKey, consumerGroup, msg.ID)
					continue
				}

				if err := applyScan(ctx, db, event); err != nil {
					log.Printf("failed to apply scan event %s: %v", msg.ID, err)
					continue
				}

				rdb.XAck(ctx, redisclient.ScanStreamKey, consumerGroup, msg.ID)
			}
		}
	}
}

func applyScan(ctx context.Context, db *pgxpool.Pool, e models.ScanEvent) error {
	tx, err := db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var oldQty int
	if err := tx.QueryRow(ctx,
		`SELECT received_quantity FROM receiving_ledger
		 WHERE invoice_id = $1 AND item_sku = $2 FOR UPDATE`,
		e.InvoiceID, e.ItemSKU).Scan(&oldQty); err != nil {
		return err
	}
	newQty := oldQty + e.Delta
	if newQty < 0 {
		newQty = 0
	}
	if _, err := tx.Exec(ctx,
		`UPDATE receiving_ledger
		 SET received_quantity = $1, last_updated = now()
		 WHERE invoice_id = $2 AND item_sku = $3`,
		newQty, e.InvoiceID, e.ItemSKU); err != nil {
		return err
	}

	if _, err := tx.Exec(ctx,
		`INSERT INTO audit_log (ts, invoice_id, item_sku, warehouse_id, user_id, action_type, old_quantity, new_quantity, delta, reason)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		e.Timestamp, e.InvoiceID, e.ItemSKU, e.WarehouseID, e.UserID, e.ActionType, oldQty, newQty, newQty-oldQty, e.Reason); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE invoices
		SET status = CASE
			WHEN totals.received >= totals.expected AND totals.expected > 0 THEN 'Completed'
			WHEN totals.received > 0 THEN 'Receiving'
			ELSE 'Pending'
		END
		FROM (
			SELECT COALESCE(sum(rl.received_quantity),0) AS received, COALESCE(sum(il.expected_quantity),0) AS expected
			FROM invoice_lines il
			LEFT JOIN receiving_ledger rl ON rl.invoice_id = il.invoice_id AND rl.item_sku = il.item_sku
			WHERE il.invoice_id = $1
		) totals
		WHERE invoices.invoice_id = $1`, e.InvoiceID); err != nil {
		return err
	}

	return tx.Commit(ctx)
}
