package handler

import (
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

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

type Penilaian struct {
	ID           string `json:"id"`
	SiswaID      string `json:"siswa_id"`
	PeriodeBulan string `json:"periode_bulan"`
	Materi       string `json:"materi"`
	NilaiAngka   int    `json:"nilai_angka"`
	Predikat     string `json:"predikat"`
}

type Ujian struct {
	ID      int64   `json:"id"`
	SiswaID string  `json:"siswa_id"`
	Periode string  `json:"periode"`
	Nilai   float64 `json:"nilai"`
	Status  string  `json:"status"`
	Catatan string  `json:"catatan"`
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

type adminLogin struct {
	Username string `json:"username"`
	Password string `json:"password"`
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
	case "/api/admin/login":
		if r.Method == http.MethodPost {
			loginAdmin(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	case "/api/admin/logout":
		logoutAdmin(w, r)
	case "/api/admin/session":
		if requireAdmin(w, r) {
			json.NewEncoder(w).Encode(map[string]string{"status": "sukses"})
		}
	case "/api/admin/siswa":
		if requireAdmin(w, r) && r.Method == http.MethodPost {
			createSiswa(w, r)
		}
	case "/api/admin/pengurus":
		if requireAdmin(w, r) && r.Method == http.MethodPost {
			createPengurus(w, r)
		}
	case "/api/admin/pelatih":
		if requireAdmin(w, r) && r.Method == http.MethodPost {
			createPelatih(w, r)
		}
	case "/api/admin/absensi":
		if requireAdmin(w, r) && r.Method == http.MethodPost {
			simpanAbsensi(w, r)
		}
	case "/api/admin/penilaian":
		if requireAdmin(w, r) && r.Method == http.MethodPost {
			createPenilaian(w, r)
		}
	case "/api/admin/ujian":
		if requireAdmin(w, r) && r.Method == http.MethodPost {
			createUjian(w, r)
		}
	case "/api/siswa":
		getSiswa(w, r)
	case "/api/absensi":
		if requireAdmin(w, r) && r.Method == http.MethodPost {
			simpanAbsensi(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	case "/api/pengurus":
		getPengurus(w, r)
	case "/api/pelatih":
		getPelatih(w, r)
	case "/api/penilaian":
		getPenilaian(w, r)
	case "/api/ujian":
		getUjian(w, r)
	default:
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"message": "Backend Go PSHT Ranting Genuk Berjalan Normal!"}`))
	}
}

func loginAdmin(w http.ResponseWriter, r *http.Request) {
	var input adminLogin
	if os.Getenv("ADMIN_USERNAME") == "" || os.Getenv("ADMIN_PASSWORD") == "" || os.Getenv("ADMIN_SESSION_SECRET") == "" || json.NewDecoder(r.Body).Decode(&input) != nil || input.Username != os.Getenv("ADMIN_USERNAME") || input.Password != os.Getenv("ADMIN_PASSWORD") {
		http.Error(w, `{"error":"Username atau password salah"}`, http.StatusUnauthorized)
		return
	}

	token := adminToken(time.Now().Unix())
	http.SetCookie(w, &http.Cookie{Name: "admin_session", Value: token, Path: "/", HttpOnly: true, Secure: true, SameSite: http.SameSiteLaxMode, MaxAge: 8 * 60 * 60})
	json.NewEncoder(w).Encode(map[string]string{"status": "sukses"})
}

func logoutAdmin(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{Name: "admin_session", Value: "", Path: "/", HttpOnly: true, Secure: true, MaxAge: -1})
	json.NewEncoder(w).Encode(map[string]string{"status": "sukses"})
}

func requireAdmin(w http.ResponseWriter, r *http.Request) bool {
	cookie, err := r.Cookie("admin_session")
	if err != nil || !validAdminToken(cookie.Value) {
		http.Error(w, `{"error":"Akses admin diperlukan"}`, http.StatusUnauthorized)
		return false
	}
	return true
}

func adminToken(timestamp int64) string {
	value := strconv.FormatInt(timestamp, 10)
	h := hmac.New(sha256.New, []byte(os.Getenv("ADMIN_SESSION_SECRET")))
	h.Write([]byte(value))
	return value + "." + hex.EncodeToString(h.Sum(nil))
}

func validAdminToken(token string) bool {
	if os.Getenv("ADMIN_SESSION_SECRET") == "" {
		return false
	}
	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		return false
	}
	timestamp, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil || time.Now().Unix()-timestamp > 8*60*60 || timestamp > time.Now().Unix() {
		return false
	}
	return hmac.Equal([]byte(parts[1]), []byte(strings.Split(adminToken(timestamp), ".")[1]))
}

func createSiswa(w http.ResponseWriter, r *http.Request) {
	var s Siswa
	if json.NewDecoder(r.Body).Decode(&s) != nil || s.Nama == "" {
		http.Error(w, `{"error":"Data siswa tidak valid"}`, http.StatusBadRequest)
		return
	}
	db, err := connectDB()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer db.Close()
	_, err = db.Exec("INSERT INTO siswa (nama, alamat, tempat_lahir, tanggal_lahir, sabuk) VALUES ($1, $2, $3, $4, $5)", s.Nama, s.Alamat, s.TempatLahir, s.TanggalLahir, s.Sabuk)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(map[string]string{"status": "sukses"})
}

func createPengurus(w http.ResponseWriter, r *http.Request) {
	var p Pengurus
	if json.NewDecoder(r.Body).Decode(&p) != nil || p.Nama == "" || p.Jabatan == "" {
		http.Error(w, `{"error":"Data pengurus tidak valid"}`, http.StatusBadRequest)
		return
	}
	db, err := connectDB()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer db.Close()
	_, err = db.Exec("INSERT INTO pengurus (jabatan, nama, keterangan) VALUES ($1, $2, $3)", p.Jabatan, p.Nama, p.Keterangan)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(map[string]string{"status": "sukses"})
}

func createPelatih(w http.ResponseWriter, r *http.Request) {
	var p Pelatih
	if json.NewDecoder(r.Body).Decode(&p) != nil || p.Nama == "" {
		http.Error(w, `{"error":"Data pelatih tidak valid"}`, http.StatusBadRequest)
		return
	}
	db, err := connectDB()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer db.Close()
	_, err = db.Exec("INSERT INTO pelatih (nama, tingkatan, spesialisasi, kontak, status) VALUES ($1, $2, $3, $4, $5)", p.Nama, p.Tingkatan, p.Spesialisasi, p.Kontak, p.Status)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(map[string]string{"status": "sukses"})
}

func createPenilaian(w http.ResponseWriter, r *http.Request) {
	var p Penilaian
	if json.NewDecoder(r.Body).Decode(&p) != nil || p.SiswaID == "" || p.PeriodeBulan == "" || p.Materi == "" || p.NilaiAngka < 0 || p.NilaiAngka > 100 || p.Predikat == "" {
		http.Error(w, `{"error":"Data penilaian tidak valid"}`, http.StatusBadRequest)
		return
	}
	db, err := connectDB()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer db.Close()
	_, err = db.Exec("INSERT INTO penilaian (siswa_id, periode_bulan, materi, nilai_angka, predikat) VALUES ($1, $2, $3, $4, $5)", p.SiswaID, p.PeriodeBulan, p.Materi, p.NilaiAngka, p.Predikat)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(map[string]string{"status": "sukses"})
}

func createUjian(w http.ResponseWriter, r *http.Request) {
	var u Ujian
	if json.NewDecoder(r.Body).Decode(&u) != nil || u.SiswaID == "" || u.Periode == "" || u.Nilai < 0 || u.Nilai > 100 || u.Status == "" {
		http.Error(w, `{"error":"Data ujian tidak valid"}`, http.StatusBadRequest)
		return
	}
	db, err := connectDB()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer db.Close()
	_, err = db.Exec("INSERT INTO ujian (siswa_id, periode, nilai, status, catatan) VALUES ($1, $2, $3, $4, $5)", u.SiswaID, u.Periode, u.Nilai, u.Status, u.Catatan)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(map[string]string{"status": "sukses"})
}

func getPenilaian(w http.ResponseWriter, r *http.Request) {
	getRekapRows(w, r, "penilaian", `SELECT id, siswa_id, periode_bulan, materi, nilai_angka, predikat FROM penilaian WHERE siswa_id = $1 ORDER BY created_at DESC`)
}

func getUjian(w http.ResponseWriter, r *http.Request) {
	getRekapRows(w, r, "ujian", `SELECT id, siswa_id, periode, nilai, status, COALESCE(catatan, '') FROM ujian WHERE siswa_id = $1 ORDER BY periode DESC, created_at DESC`)
}

func getRekapRows(w http.ResponseWriter, r *http.Request, table string, query string) {
	siswaID := r.URL.Query().Get("siswa_id")
	if siswaID == "" {
		http.Error(w, `{"error":"siswa_id wajib diisi"}`, http.StatusBadRequest)
		return
	}
	db, err := connectDB()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer db.Close()
	rows, err := db.Query(query, siswaID)
	if err != nil {
		http.Error(w, "Gagal mengambil rekap "+table, http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	if table == "penilaian" {
		results := make([]Penilaian, 0)
		for rows.Next() {
			var p Penilaian
			if err := rows.Scan(&p.ID, &p.SiswaID, &p.PeriodeBulan, &p.Materi, &p.NilaiAngka, &p.Predikat); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			results = append(results, p)
		}
		json.NewEncoder(w).Encode(results)
		return
	}
	results := make([]Ujian, 0)
	for rows.Next() {
		var u Ujian
		if err := rows.Scan(&u.ID, &u.SiswaID, &u.Periode, &u.Nilai, &u.Status, &u.Catatan); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		results = append(results, u)
	}
	json.NewEncoder(w).Encode(results)
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
