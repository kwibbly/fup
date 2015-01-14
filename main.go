package main

import (
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

type page struct {
	Title string
	Body  []byte
}

type Download struct {
	Id         int       `db:"id"`
	Filename   string    `db:"filename"`
	UploadDate time.Time `db:"uploadDate"`
}

type DownloadPage struct {
	Downloads []Download
	Title     string
}

func init() {
	os.Mkdir("./downloads", 0755)
	initDB()
}

func main() {
	db, err := sqlx.Open("sqlite3", "./fup.db")
	if err != nil {
		log.Fatal("Something wrong with my db: ", err)
	}
	defer db.Close()

	http.Handle("/assets/", http.StripPrefix("/assets/", http.FileServer(http.Dir("./assets"))))
	http.Handle("/_downloads/", http.StripPrefix("/_downloads/", http.FileServer(http.Dir("./downloads"))))
	http.HandleFunc("/upload", uploadHandler)
	http.HandleFunc("/downloads/", downloadHandler)
	http.HandleFunc("/", doRest)
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func initDB() {
	if _, err := os.Stat("./fup.db"); err != nil {
		db, _ := sqlx.Open("sqlite3", "./fup.db")
		sql := `
		CREATE TABLE uploads (id integer not null primary key, filename text UNIQUE, uploadDate timestamp);
		`
		db.Exec(sql)
		db.Close()
		log.Println("Successfully created new DB")
	}
}

func uploadHandler(w http.ResponseWriter, r *http.Request) {
	db, err := sqlx.Open("sqlite3", "./fup.db")
	if err != nil {
		log.Fatal("Something wrong with my db: ", err)
	}
	defer db.Close()
	tx, _ := db.Begin()
	defer tx.Commit()

	file, header, err := r.FormFile("file")
	if err != nil {
		log.Println("Error while uploading: ", err.Error())
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}
	if header.Filename == "index.html" {
		log.Println("somebody tried to upload an index.html, dropping request")
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}
	defer file.Close()
	out, err := os.Create("./downloads/" + header.Filename)
	defer out.Close()
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		log.Println("Error while creating file: ", err.Error())
	}
	_, err = io.Copy(out, file)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		log.Println("Error while writing to file: ", err.Error())
	}

	cTime := time.Now()
	sql := `
	INSERT INTO uploads (filename, uploadDate) VALUES (?,?);
	`
	db.Exec(sql, header.Filename, cTime)

	http.Redirect(w, r, "/downloads", http.StatusFound)

}

func downloadHandler(w http.ResponseWriter, r *http.Request) {
	db, err := sqlx.Open("sqlite3", "./fup.db")
	if err != nil {
		log.Fatal("Something wrong with my db: ", err)
	}
	defer db.Close()

	files := []Download{}
	err = db.Select(&files, "SELECT id,filename,uploadDate FROM uploads;")
	if err != nil {
		log.Println("Something wrong with the db: ", err)
	}

	dp := &DownloadPage{Title: "File Downloads", Downloads: files}
	t, _ := template.ParseFiles("assets/download.html")
	t.Execute(w, dp)
}

func doRest(w http.ResponseWriter, r *http.Request) {
	p := &page{Title: "File UPload"}
	t, _ := template.ParseFiles("assets/index.html")
	t.Execute(w, p)
}
