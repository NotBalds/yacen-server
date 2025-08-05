package main

import (
	"fmt"
	"os"
	"time"

	api "github.com/NotBalds/yacen-server/yacen_api.v2_2"
	"github.com/charmbracelet/log"
	"github.com/lib/pq"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type Room struct {
	gorm.Model
	RID string
	// PrivateInfo
	AdminKeys           pq.StringArray `gorm:"type:text[]"`
	AllowedKeys         pq.StringArray `gorm:"type:text[]"`
	PendingJoinRequests pq.StringArray `gorm:"type:text[]"`
	// PublicInfo
	Type          api.RoomType
	EncryptedName []byte `gorm:"type:bytea"`
	EncryptedDesc []byte `gorm:"type:bytea"`
}

func newDB() *gorm.DB {
	user := os.Getenv("POSTGRES_USER")
	pass := os.Getenv("POSTGRES_PASSWORD")
	host := os.Getenv("POSTGRES_HOST")

	var dsn string

	if host == "" {
		dsn = fmt.Sprintf("host=localhost user=%s dbname=yacen-server-db password=%s port=5432 sslmode=disable", user, pass)
	} else {
		dsn = fmt.Sprintf("host=%s user=%s dbname=yacen-server-db password=%s port=5432 sslmode=disable", host, user, pass)
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	db.AutoMigrate(&Room{})

	sqlDB, err := db.DB()
	if err != nil {
		log.Fatal("Failed to get generic database object:", err)
	}

	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)
	sqlDB.SetConnMaxIdleTime(30 * time.Minute)

	if err := sqlDB.Ping(); err != nil {
		log.Fatal("Database ping failed:", err)
	}

	return db
}
