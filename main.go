// falta que el GET sea 1. estado ACTUAL!! y 2. solo status

package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type Payment struct {
	EventID   string    `gorm:"primaryKey" json:"event_id" binding:"required"`
	PaymentID string    `json:"payment_id" binding:"required"`
	Amount    uint      `json:"amount" binding:"required"`
	Status    string    `json:"status" binding:"required"`
	Timestamp time.Time `json:"timestamp" binding:"required"`
}

// syncSheet replaces the whole sheet with the current contents of the database,
// using the same columns as the Payment table.
func syncSheet(ctx context.Context, db *gorm.DB, srv *sheets.Service, spreadsheetID, sheetName string) error {
	payments, err := gorm.G[Payment](db).Order("timestamp ASC").Order("event_id ASC").Find(ctx)
	if err != nil {
		return fmt.Errorf("failed to read payments from database: %w", err)
	}

	values := make([][]interface{}, 0, len(payments)+1)
	values = append(values, []interface{}{"event_id", "payment_id", "amount", "status", "timestamp"})
	for _, p := range payments {
		values = append(values, []interface{}{
			p.EventID,
			p.PaymentID,
			p.Amount,
			p.Status,
			p.Timestamp.Format(time.RFC3339),
		})
	}

	quotedName := "'" + sheetName + "'"

	if _, err := srv.Spreadsheets.Values.Clear(spreadsheetID, quotedName+"!A:Z", &sheets.ClearValuesRequest{}).Do(); err != nil {
		return fmt.Errorf("failed to clear sheet: %w", err)
	}

	valueRange := &sheets.ValueRange{Values: values}
	if _, err := srv.Spreadsheets.Values.Update(spreadsheetID, quotedName+"!A1", valueRange).ValueInputOption("RAW").Do(); err != nil {
		return fmt.Errorf("failed to write sheet: %w", err)
	}

	return nil
}

func main() {

	if err := godotenv.Load(); err != nil {
		log.Println("no .env file, using environment")
	}

	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		os.Getenv("DB_HOST"), os.Getenv("DB_PORT"), os.Getenv("DB_USER"), os.Getenv("DB_PASSWORD"), os.Getenv("DB_NAME"))
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{TranslateError: true})
	if err != nil {
		panic("failed to connect database")
	}

	db.AutoMigrate(&Payment{})

	spreadsheetID := os.Getenv("SPREADSHEET_ID")
	sheetName := os.Getenv("SHEET_NAME")
	credsFile := os.Getenv("GOOGLE_CREDENTIALS_FILE")

	sheetsSrv, err := sheets.NewService(context.Background(), option.WithCredentialsFile(credsFile))
	if err != nil {
		log.Fatalf("unable to retrieve Sheets client: %v", err)
	}

	// Create a Gin router with default middleware (logger and recovery)
	r := gin.Default()

	r.POST("/webhooks/payments", func(c *gin.Context) {
		var payment Payment

		if err := c.ShouldBindJSON(&payment); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		err = gorm.G[Payment](db).Create(c, &payment)
		if err != nil {
			if errors.Is(err, gorm.ErrDuplicatedKey) {
				log.Print("ERROR: Duplicated event on payment ", payment)
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			}
			return

		}

		if err := syncSheet(c, db, sheetsSrv, spreadsheetID, sheetName); err != nil {
			log.Printf("ERROR: failed to sync sheet: %v", err)
			c.JSON(http.StatusOK, gin.H{"payment": payment, "sheet_sync": "failed", "error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, payment)
	})

	r.GET("/payments/:id", func(c *gin.Context) {
		id := c.Param("id")

		payment, err := gorm.G[Payment](db).Where("payment_id = ?", id).Order("timestamp DESC").First(c)

		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, payment)

	})

	// Start server on port 80
	// Server will listen on 0.0.0.0:80 (localhost:80 on Windows)
	if err := r.Run(":80"); err != nil {
		log.Fatalf("failed to run server: %v", err)
	}
}
