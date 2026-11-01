package database

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"modul10/config"
)

// NewMongoClient membuat dan menguji koneksi ke server MongoDB.
// Menggunakan driver go.mongodb.org/mongo-driver/v2.
// Wajib memanggil Ping agar aplikasi menolak menyala jika MongoDB tidak dapat dihubungi.
func NewMongoClient(ctx context.Context) (*mongo.Client, *mongo.Database, error) {
	uri := config.GetEnv("MONGO_URI", "mongodb://127.0.0.1:27017")
	dbName := config.GetEnv("MONGO_DB", "praktikum_backend")

	clientOptions := options.Client().
		ApplyURI(uri).
		SetConnectTimeout(5 * time.Second).
		SetServerSelectionTimeout(5 * time.Second)

	client, err := mongo.Connect(clientOptions)
	if err != nil {
		return nil, nil, fmt.Errorf("gagal membuat client mongodb: %w", err)
	}

	// Ping wajib untuk memastikan koneksi fisik aktif
	pingCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	if err := client.Ping(pingCtx, nil); err != nil {
		_ = client.Disconnect(ctx)
		return nil, nil, fmt.Errorf("gagal terhubung ke mongodb: %w", err)
	}

	db := client.Database(dbName)
	return client, db, nil
}
