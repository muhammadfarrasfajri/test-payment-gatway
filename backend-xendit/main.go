package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/muhammadfarrasfajri/test-payment-gatway.git/db"
)

var (
	XenditSecretKey = os.Getenv("PRIVATE_KEY")
	XenditToken     = os.Getenv("TOKEN")
)

type CheckoutRequest struct {
	Amount        float64 `json:"amount" binding:"required"`
	PaymentMethod string  `json:"payment_method" binding:"required"`
	CustomerEmail string  `json:"customer_email" binding:"required"`
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	// 1. Inisialisasi koneksi PostgreSQL
	db.Connect()

	r := gin.Default()

	// 2. Middleware CORS agar Next.js (localhost:3000) bisa mengakses API ini
	r.Use(cors.New(cors.Config{
		AllowOrigins: []string{"http://localhost:3001"},
		AllowMethods: []string{"POST", "GET", "OPTIONS"},
		AllowHeaders: []string{"Origin", "Content-Type", "Accept", "x-callback-token"},
	}))

	// --- ENDPOINT 1: CHECKOUT ---
	r.POST("/api/checkout", func(c *gin.Context) {
		var req CheckoutRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Buat Order ID Unik
		orderID := fmt.Sprintf("ORDER-%d", time.Now().Unix())

		// Buat tagihan ke Xendit
		invoiceURL, err := createXenditInvoice(orderID, req.Amount, req.PaymentMethod, req.CustomerEmail)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membuat invoice Xendit"})
			return
		}

		// Insert data ke PostgreSQL (Status awal: PENDING)
		query := `
			INSERT INTO transactions (order_id, amount, payment_method, status, invoice_url, created_at)
			VALUES ($1, $2, $3, $4, $5, $6)
		`
		_, err = db.DB.Exec(query, orderID, req.Amount, req.PaymentMethod, "PENDING", invoiceURL, time.Now())
		if err != nil {
			log.Printf("Gagal insert database: %v\n", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan transaksi"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message":     "Checkout berhasil",
			"order_id":    orderID,
			"invoice_url": invoiceURL,
			"status":      "PENDING",
		})
	})

	// --- ENDPOINT 2: WEBHOOK XENDIT ---
	r.POST("/webhook/xendit/invoice", func(c *gin.Context) {
		// Validasi Token Keamanan dari Xendit
		if c.GetHeader("x-callback-token") != XenditToken {
			c.JSON(http.StatusUnauthorized, gin.H{"message": "unauthorized"})
			return
		}

		var payload map[string]interface{}
		if err := c.ShouldBindJSON(&payload); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		externalID := payload["external_id"].(string)
		status := payload["status"].(string)

		var newStatus string
		if status == "PAID" || status == "SETTLED" {
			newStatus = "PAID"
		} else if status == "EXPIRED" {
			newStatus = "EXPIRED"
		} else {
			c.JSON(http.StatusOK, gin.H{"status": "ignored"})
			return
		}

		// Mulai Database Transaction
		tx, err := db.DB.Begin()
		if err != nil {
			log.Printf("Gagal memulai transaksi: %v\n", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
			return
		}
		// Defer rollback akan dieksekusi jika fungsi return sebelum Commit
		defer tx.Rollback()

		// Ambil data dengan ROW-LEVEL LOCKING (FOR UPDATE)
		var currentStatus string
		querySelect := `SELECT status FROM transactions WHERE order_id = $1 FOR UPDATE`

		err = tx.QueryRow(querySelect, externalID).Scan(&currentStatus)
		if err != nil {
			if err == sql.ErrNoRows {
				c.JSON(http.StatusNotFound, gin.H{"message": "Order tidak ditemukan di database"})
				return
			}
			log.Printf("Error saat melock row: %v\n", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
			return
		}

		// Idempotency (Mencegah Double Processing / Race Condition)
		if currentStatus == "PAID" || currentStatus == "EXPIRED" {
			fmt.Printf("⚠️ Webhook duplikat terdeteksi untuk Order %s. Status sudah %s. Diabaikan.\n", externalID, currentStatus)
			c.JSON(http.StatusOK, gin.H{"status": "already_processed"})
			return
		}

		// Update status transaksi
		queryUpdate := `UPDATE transactions SET status = $1 WHERE order_id = $2`
		_, err = tx.Exec(queryUpdate, newStatus, externalID)
		if err != nil {
			log.Printf("Gagal update database: %v\n", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal update transaksi"})
			return
		}

		// Commit Transaksi (Menyimpan perubahan dan melepas Lock)
		if err := tx.Commit(); err != nil {
			log.Printf("Gagal commit transaksi: %v\n", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
			return
		}

		fmt.Printf("✅ Order %s berhasil diupdate menjadi %s\n", externalID, newStatus)
		c.JSON(http.StatusOK, gin.H{"status": "success"})
	})

	fmt.Println("Server Backend berjalan di http://localhost:8080")
	r.Run(":8080")
}

// --- HELPER FUNCTION: Menembak API Xendit ---
func createXenditInvoice(orderID string, amount float64, method string, email string) (string, error) {
	url := "https://api.xendit.co/v2/invoices"

	// Kita pass payment_methods sebagai array agar Xendit HANYA menampilkan metode yang dipilih user
	payload := map[string]interface{}{
		"external_id":     orderID,
		"amount":          amount,
		"description":     "Pembayaran Pesanan " + orderID,
		"payer_email":     email,
		"payment_methods": []string{method},
	}

	jsonData, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))

	req.SetBasicAuth(XenditSecretKey, "")
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	// Kembalikan invoice_url
	if invoiceURL, ok := result["invoice_url"].(string); ok {
		return invoiceURL, nil
	}

	return "", fmt.Errorf("gagal mendapatkan invoice url: %v", result)
}
