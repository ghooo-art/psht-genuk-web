package handler

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
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
	FotoURL      string `json:"foto_url"`
}

// Struktur data Absensi
type Absensi struct {
	SiswaID string `json:"siswa_id"`
	Tanggal string `json:"tanggal"`
	Status  string `json:"status"` // 'hadir', 'izin', 'alpha'
}

type RekapAbsensi struct {
	SiswaID string  `json:"siswa_id"`
	Hadir   int     `json:"hadir"`
	Izin    int     `json:"izin"`
	Alpha   int     `json:"alpha"`
	Persen  float64 `json:"persentase"`
	Sesi    int     `json:"sesi"`
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
	FotoURL    string `json:"foto_url"`
}

// Struktur data Pelatih
type Pelatih struct {
	ID           int    `json:"id"`
	Nama         string `json:"nama"`
	Tingkatan    string `json:"tingkatan"`
	Spesialisasi string `json:"spesialisasi"`
	Kontak       string `json:"kontak"`
	Status       string `json:"status"`
	FotoURL      string `json:"foto_url"`
}

type JadwalLatihan struct {
	ID    int    `json:"id"`
	Hari  string `json:"hari"`
	Waktu string `json:"waktu"`
	Aktif bool   `json:"aktif"`
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
	case "/api/admin/upload":
		if requireAdmin(w, r) && r.Method == http.MethodPost {
			uploadFoto(w, r)
		}
	case "/api/admin/jadwal":
		if requireAdmin(w, r) && r.Method == http.MethodPost {
			simpanJadwal(w, r)
		}
	case "/api/siswa":
		getSiswa(w, r)
	case "/api/absensi":
		if r.Method == http.MethodGet {
			getAbsensiBulan(w, r)
		} else if requireAdmin(w, r) && r.Method == http.MethodPost {
			simpanAbsensi(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	case "/api/pengurus":
		getPengurus(w, r)
	case "/api/pelatih":
		getPelatih(w, r)
	case "/api/jadwal":
		getJadwal(w, r)
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
	if json.NewDecoder(r.Body).Decode(&s) != nil || s.Nama == "" || s.Alamat == "" || s.TempatLahir == "" || s.TanggalLahir == "" || s.Sabuk == "" {
		http.Error(w, `{"error":"Data siswa tidak valid"}`, http.StatusBadRequest)
		return
	}
	db, err := connectDB()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer db.Close()
	_, err = db.Exec("INSERT INTO siswa (nama, alamat, tempat_lahir, tanggal_lahir, sabuk, foto_url) VALUES ($1, $2, $3, $4, $5, $6)", s.Nama, s.Alamat, s.TempatLahir, s.TanggalLahir, s.Sabuk, s.FotoURL)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(map[string]string{"status": "sukses"})
}

func uploadFoto(w http.ResponseWriter, r *http.Request) {
	if os.Getenv("SUPABASE_URL") == "" || os.Getenv("SUPABASE_SERVICE_ROLE_KEY") == "" {
		http.Error(w, `{"error":"Konfigurasi Supabase Storage belum tersedia"}`, http.StatusInternalServerError)
		return
	}
	if err := r.ParseMultipartForm(6 << 20); err != nil {
		http.Error(w, `{"error":"Ukuran foto maksimal 5 MB"}`, http.StatusBadRequest)
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, `{"error":"File foto wajib dipilih"}`, http.StatusBadRequest)
		return
	}
	defer file.Close()

	contentType := header.Header.Get("Content-Type")
	if contentType != "image/jpeg" && contentType != "image/png" && contentType != "image/webp" {
		http.Error(w, `{"error":"Format foto harus JPG, PNG, atau WEBP"}`, http.StatusBadRequest)
		return
	}
	data, err := io.ReadAll(io.LimitReader(file, 5<<20+1))
	if err != nil || len(data) > 5<<20 {
		http.Error(w, `{"error":"Ukuran foto maksimal 5 MB"}`, http.StatusBadRequest)
		return
	}

	category := r.FormValue("category")
	if category != "siswa" && category != "pelatih" && category != "pengurus" {
		http.Error(w, `{"error":"Kategori foto tidak valid"}`, http.StatusBadRequest)
		return
	}
	ext := filepath.Ext(header.Filename)
	path := fmt.Sprintf("%s/%d%s", category, time.Now().UnixNano(), ext)
	uploadURL := fmt.Sprintf("%s/storage/v1/object/psht-foto/%s", strings.TrimRight(os.Getenv("SUPABASE_URL"), "/"), path)
	req, err := http.NewRequest(http.MethodPost, uploadURL, bytes.NewReader(data))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	req.Header.Set("Authorization", "Bearer "+os.Getenv("SUPABASE_SERVICE_ROLE_KEY"))
	req.Header.Set("apikey", os.Getenv("SUPABASE_SERVICE_ROLE_KEY"))
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("x-upsert", "true")
	response, err := http.DefaultClient.Do(req)
	if err != nil || response.StatusCode < 200 || response.StatusCode >= 300 {
		if response != nil {
			response.Body.Close()
		}
		http.Error(w, `{"error":"Gagal mengunggah foto ke Supabase Storage"}`, http.StatusBadGateway)
		return
	}
	response.Body.Close()
	publicURL := fmt.Sprintf("%s/storage/v1/object/public/psht-foto/%s", strings.TrimRight(os.Getenv("SUPABASE_URL"), "/"), path)
	json.NewEncoder(w).Encode(map[string]string{"foto_url": publicURL})
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
	_, err = db.Exec("INSERT INTO pengurus (jabatan, nama, keterangan, foto_url) VALUES ($1, $2, $3, $4)", p.Jabatan, p.Nama, p.Keterangan, p.FotoURL)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(map[string]string{"status": "sukses"})
}

func createPelatih(w http.ResponseWriter, r *http.Request) {
	var p Pelatih
	if json.NewDecoder(r.Body).Decode(&p) != nil || p.Nama == "" || p.Tingkatan == "" || p.Spesialisasi == "" || p.Kontak == "" {
		http.Error(w, `{"error":"Data pelatih tidak valid"}`, http.StatusBadRequest)
		return
	}
	db, err := connectDB()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer db.Close()
	_, err = db.Exec("INSERT INTO pelatih (nama, tingkatan, spesialisasi, kontak, status, foto_url) VALUES ($1, $2, $3, $4, $5, $6)", p.Nama, p.Tingkatan, p.Spesialisasi, p.Kontak, p.Status, p.FotoURL)
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

func getRekapAbsensi(w http.ResponseWriter, db *sql.DB, start, end time.Time) {
	rows, err := db.Query(`SELECT s.id,
		COUNT(a.*) FILTER (WHERE a.status = 'hadir'),
		COUNT(a.*) FILTER (WHERE a.status = 'izin'),
		COUNT(a.*) FILTER (WHERE a.status = 'alpha'),
		COUNT(DISTINCT a.tanggal)
		FROM siswa s LEFT JOIN absensi a ON a.siswa_id = s.id AND a.tanggal >= $1 AND a.tanggal < $2
		GROUP BY s.id ORDER BY s.nama`, start, end)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	results := make([]RekapAbsensi, 0)
	for rows.Next() {
		var item RekapAbsensi
		if err := rows.Scan(&item.SiswaID, &item.Hadir, &item.Izin, &item.Alpha, &item.Sesi); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if item.Sesi > 0 {
			item.Persen = float64(item.Hadir) / float64(item.Sesi) * 100
		}
		results = append(results, item)
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

	rows, err := db.Query("SELECT id, nama, alamat, tempat_lahir, tanggal_lahir, sabuk, foto_url FROM siswa")
	if err != nil {
		http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var listSiswa []Siswa
	for rows.Next() {
		var s Siswa
		if err := rows.Scan(&s.ID, &s.Nama, &s.Alamat, &s.TempatLahir, &s.TanggalLahir, &s.Sabuk, &s.FotoURL); err != nil {
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
	if err != nil || abs.SiswaID == "" || abs.Tanggal == "" || (abs.Status != "hadir" && abs.Status != "izin" && abs.Status != "alpha") {
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

func getAbsensiBulan(w http.ResponseWriter, r *http.Request) {
	bulan := r.URL.Query().Get("bulan")
	start, err := time.Parse("2006-01", bulan)
	if err != nil {
		http.Error(w, `{"error":"Format bulan harus YYYY-MM"}`, http.StatusBadRequest)
		return
	}
	end := start.AddDate(0, 1, 0)
	db, err := connectDB()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer db.Close()
	if r.URL.Query().Get("rekap") == "true" {
		getRekapAbsensi(w, db, start, end)
		return
	}

	rows, err := db.Query(`SELECT tanggal, CASE
		WHEN BOOL_OR(status = 'hadir') THEN 'hadir'
		WHEN BOOL_OR(status = 'izin') THEN 'izin'
		ELSE 'alpha' END AS status
		FROM absensi WHERE tanggal >= $1 AND tanggal < $2 GROUP BY tanggal ORDER BY tanggal`, start, end)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	results := make([]map[string]string, 0)
	for rows.Next() {
		var tanggal time.Time
		var status string
		if err := rows.Scan(&tanggal, &status); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		results = append(results, map[string]string{"tanggal": tanggal.Format("2006-01-02"), "status": status})
	}
	json.NewEncoder(w).Encode(results)
}

// 3. Endpoint GET: Mengambil data pengurus ranting
func getPengurus(w http.ResponseWriter, r *http.Request) {
	db, err := connectDB()
	if err != nil {
		http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}
	defer db.Close()

	rows, err := db.Query("SELECT id, jabatan, nama, keterangan, foto_url FROM pengurus")
	if err != nil {
		http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var listPengurus []Pengurus
	for rows.Next() {
		var p Pengurus
		if err := rows.Scan(&p.ID, &p.Jabatan, &p.Nama, &p.Keterangan, &p.FotoURL); err != nil {
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

	rows, err := db.Query("SELECT id, nama, tingkatan, spesialisasi, kontak, status, foto_url FROM pelatih")
	if err != nil {
		http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var listPelatih []Pelatih
	for rows.Next() {
		var l Pelatih
		if err := rows.Scan(&l.ID, &l.Nama, &l.Tingkatan, &l.Spesialisasi, &l.Kontak, &l.Status, &l.FotoURL); err != nil {
			continue
		}
		listPelatih = append(listPelatih, l)
	}

	json.NewEncoder(w).Encode(listPelatih)
}

func ensureJadwalTable(db *sql.DB) error {
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS jadwal_latihan (
		id SERIAL PRIMARY KEY,
		hari TEXT NOT NULL,
		waktu TEXT NOT NULL,
		aktif BOOLEAN NOT NULL DEFAULT TRUE
	)`)
	return err
}

func getJadwal(w http.ResponseWriter, r *http.Request) {
	db, err := connectDB()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer db.Close()
	if err := ensureJadwalTable(db); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM jadwal_latihan WHERE aktif = TRUE").Scan(&count); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if count == 0 {
		_, err = db.Exec("INSERT INTO jadwal_latihan (hari, waktu) VALUES ($1, $2), ($3, $4)", "Selasa", "19:22", "Sabtu", "19:22")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	rows, err := db.Query("SELECT id, hari, waktu, aktif FROM jadwal_latihan WHERE aktif = TRUE ORDER BY id")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	jadwal := make([]JadwalLatihan, 0)
	for rows.Next() {
		var item JadwalLatihan
		if err := rows.Scan(&item.ID, &item.Hari, &item.Waktu, &item.Aktif); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		jadwal = append(jadwal, item)
	}
	json.NewEncoder(w).Encode(jadwal)
}

func simpanJadwal(w http.ResponseWriter, r *http.Request) {
	var jadwal []JadwalLatihan
	if json.NewDecoder(r.Body).Decode(&jadwal) != nil || len(jadwal) == 0 || len(jadwal) > 7 {
		http.Error(w, `{"error":"Minimal satu jadwal dan maksimal tujuh jadwal"}`, http.StatusBadRequest)
		return
	}
	for _, item := range jadwal {
		if strings.TrimSpace(item.Hari) == "" || strings.TrimSpace(item.Waktu) == "" {
			http.Error(w, `{"error":"Hari dan waktu wajib diisi"}`, http.StatusBadRequest)
			return
		}
	}

	db, err := connectDB()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer db.Close()
	if err := ensureJadwalTable(db); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	tx, err := db.Begin()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if _, err = tx.Exec("DELETE FROM jadwal_latihan"); err != nil {
		tx.Rollback()
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	for _, item := range jadwal {
		if _, err = tx.Exec("INSERT INTO jadwal_latihan (hari, waktu, aktif) VALUES ($1, $2, TRUE)", strings.TrimSpace(item.Hari), strings.TrimSpace(item.Waktu)); err != nil {
			tx.Rollback()
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	if err = tx.Commit(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(map[string]string{"status": "sukses"})
}

// Fungsi bantu koneksi ke database Supabase
func connectDB() (*sql.DB, error) {
	connStr := os.Getenv("DATABASE_URL")
	return sql.Open("postgres", connStr)
}
