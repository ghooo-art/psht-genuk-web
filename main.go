package main

import (
    "net/http"
    
    // Ganti "psht-genuk-web/api" sesuai dengan baris pertama file go.mod Anda
    _ "psht-genuk-web/api" 
)

func main() {
    // Memanggil handler agar impor tidak sia-sia dan lolos dari error Go compiler
    http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        w.Write([]byte("PSHT Ranting Genuk Backend Running"))
    })
}