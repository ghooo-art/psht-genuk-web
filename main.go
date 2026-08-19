package main

import (
	"log"
	"net/http"
	"os"

	handler "psht-genuk-web/api" // Sesuaikan dengan nama modul di go.mod Anda

	"github.com/joho/godotenv"
)

func main() {
	// Memuat file .env jika ada
	_ = godotenv.Load()

	if os.Getenv("DATABASE_URL") == "" {
		log.Fatal("DATABASE_URL belum diset di environment atau .env")
	}

	// Mendaftarkan endpoint API khusus
	http.HandleFunc("/api/siswa", handler.Handler)
	http.HandleFunc("/api/absensi", handler.Handler)

	// Melayani file statis frontend dari folder public dengan prefix / (hanya untuk file web)
	fs := http.FileServer(http.Dir("./public"))
	http.Handle("/", fs)

	log.Println("Server lokal berjalan di: http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}