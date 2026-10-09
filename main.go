package main

import (
  "fmt"
  "log"
  "net/http"
  "os"

  "gorm.io/driver/postgres"
  "gorm.io/gorm"
  "github.com/gin-gonic/gin"
)

type Payment struct {
	EventID string			`gorm:"primaryKey" json:"event_id" binding:"required"`
	PaymentID string		`json:"payment_id" binding:"required"`
	Amount uint				`json:"amount" binding:"required"`
	Status string			`json:"status" binding:"required"`
	Timestamp string		`json:"timestamp" binding:"required"`
}


func main() {

  dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
    os.Getenv("DB_HOST"), os.Getenv("DB_PORT"), os.Getenv("DB_USER"), os.Getenv("DB_PASSWORD"), os.Getenv("DB_NAME"))
  db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
  if err != nil {
    panic("failed to connect database")
  }

  db.AutoMigrate(&Payment{})

  // Create a Gin router with default middleware (logger and recovery)
  r := gin.Default()

  // Define a simple GET endpoint
  r.GET("/ping", func(c *gin.Context) {
    // Return JSON response
    c.JSON (http.StatusOK, gin.H{
      "message": "pong",
    })
  })

  r.POST("/webhooks/payments", func(c *gin.Context) {
		var payment Payment

		if err := c.ShouldBindJSON(&payment); err != nil {
      c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
      return
		}

    err = gorm.G[Payment](db).Create(c, &payment)
		c.JSON(http.StatusOK, gin.H{
			"message": "OK",
		})
  })


  // Start server on port 8080 (default)
  // Server will listen on 0.0.0.0:8080 (localhost:8080 on Windows)
  if err := r.Run(); err != nil {
    log.Fatalf("failed to run server: %v", err)
  }
}
