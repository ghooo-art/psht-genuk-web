package handler

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"os"

	_ "github.com/lib/pq"
)

// Struktur data Siswa
type Siswa struct {
	ID           string `json:"id"`
	Nama         string `json:"nama"`
	Alamat       string `json:"alamat"`
	TempatLahir  string `json:"tempat_lahir"`
	TanggalLahir string `json:"tanggal_lahir"`
	Sabuk        string `json:"sabuk"`
}

// Struktur data Absensi
type Absensi struct {
	SiswaID string `json:"siswa_id"`
	Tanggal string `json:"tanggal"`
	Status  string `json:"status"` // 'hadir', 'izin', 'alpha'
}

// Struktur data Pengurus Ranting
type Pengurus struct {
	ID         int    `json:"id"`
	Jabatan    string `json:"jabatan"`
	Nama       string `json:"nama"`
	Keterangan string `json:"keterangan"`
}

// Struktur data Pelatih
type Pelatih struct {
	ID           int    `json:"id"`
	Nama         string `json:"nama"`
	Tingkatan    string `json:"tingkatan"`
	Spesialisasi string `json:"spesialisasi"`
	Kontak       string `json:"kontak"`
	Status       string `json:"status"`
}

func Handler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	// Handle preflight CORS request
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	// Routing berdasarkan Path URL
	switch r.URL.Path {
	case "/api/siswa":
		getSiswa(w, r)
	case "/api/absensi":
		if r.Method == http.MethodPost {
			simpanAbsensi(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	case "/api/pengurus":
		getPengurus(w, r)
	case "/api/pelatih":
		getPelatih(w, r)
	default:
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"message": "Backend Go PSHT Ranting Genuk Berjalan Normal!"}`))
	}
}

// 1. Endpoint GET: Mengambil seluruh data siswa dari Supabase
func getSiswa(w http.ResponseWriter, r *http.Request) {
	db, err := connectDB()
	if err != nil {
		http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}
	defer db.Close()

	rows, err := db.Query("SELECT id, nama, alamat, tempat_lahir, tanggal_lahir, sabuk FROM siswa")
	if err != nil {
		http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var listSiswa []Siswa
	for rows.Next() {
		var s Siswa
		if err := rows.Scan(&s.ID, &s.Nama, &s.Alamat, &s.TempatLahir, &s.TanggalLahir, &s.Sabuk); err != nil {
			continue
		}
		listSiswa = append(listSiswa, s)
	}

	json.NewEncoder(w).Encode(listSiswa)
}

// 2. Endpoint POST: Menyimpan data absensi harian
func simpanAbsensi(w http.ResponseWriter, r *http.Request) {
	db, err := connectDB()
	if err != nil {
		http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}
	defer db.Close()

	var abs Absensi
	err = json.NewDecoder(r.Body).Decode(&abs)
	if err != nil {
		http.Error(w, `{"error": "Format data tidak valid"}`, http.StatusBadRequest)
		return
	}

	_, err = db.Exec("INSERT INTO absensi (siswa_id, tanggal, status) VALUES ($1, $2, $3)", 
		abs.SiswaID, abs.Tanggal, abs.Status)
	if err != nil {
		http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "sukses", "message": "Absensi berhasil disimpan"})
}

// 3. Endpoint GET: Mengambil data pengurus ranting
func getPengurus(w http.ResponseWriter, r *http.Request) {
	db, err := connectDB()
	if err != nil {
		http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}
	defer db.Close()

	rows, err := db.Query("SELECT id, jabatan, nama, keterangan FROM pengurus")
	if err != nil {
		http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var listPengurus []Pengurus
	for rows.Next() {
		var p Pengurus
		if err := rows.Scan(&p.ID, &p.Jabatan, &p.Nama, &p.Keterangan); err != nil {
			continue
		}
		listPengurus = append(listPengurus, p)
	}

	json.NewEncoder(w).Encode(listPengurus)
}

// 4. Endpoint GET: Mengambil data pelatih ranting
func getPelatih(w http.ResponseWriter, r *http.Request) {
	db, err := connectDB()
	if err != nil {
		http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}
	defer db.Close()

	rows, err := db.Query("SELECT id, nama, tingkatan, spesialisasi, kontak, status FROM pelatih")
	if err != nil {
		http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var listPelatih []Pelatih
	for rows.Next() {
		var l Pelatih
		if err := rows.Scan(&l.ID, &l.Nama, &l.Tingkatan, &l.Spesialisasi, &l.Kontak, &l.Status); err != nil {
			continue
		}
		listPelatih = append(listPelatih, l)
	}

	json.NewEncoder(w).Encode(listPelatih)
}

// Fungsi bantu koneksi ke database Supabase
func connectDB() (*sql.DB, error) {
	connStr := os.Getenv("DATABASE_URL")
	return sql.Open("postgres", connStr)
}