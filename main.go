// Package fup provides a simple mechanism to share files.
//
// Have you ever been to a lan-party where you needed to
// share some files with your buddies?
// Theres probably always windows filesharing, however that
// does not always work, maybe there is someone with OS X
// or FreeBSD.
// fup saves the day, it provides a fast way to share files
// via HTTP.
package main

import (
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

type indexPage struct {
	Title string
	Body  []byte
}

type download struct {
	Id         int       `db:"id"`
	Filename   string    `db:"filename"`
	UploadDate time.Time `db:"uploadDate"`
}

type downloadPage struct {
	Downloads []download
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

	// scan downloads directory on startup and commit files found to the DB.
	filepath.Walk("./downloads", visitFile)

	http.Handle("/assets/", http.StripPrefix("/assets/", http.FileServer(http.Dir("./assets"))))
	http.Handle("/_downloads/", http.StripPrefix("/_downloads/", http.FileServer(http.Dir("./downloads"))))
	http.HandleFunc("/upload", uploadHandler)
	http.HandleFunc("/downloads/", downloadHandler)
	http.HandleFunc("/rescan", rescanHandler)
	http.HandleFunc("/", doRest)
	log.Fatal(http.ListenAndServe(":8080", nil))
}

// initializes the DB if it does not already exist
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

// commitFile does stuff
func visitFile(path string, f os.FileInfo, err error) error {
	db, err := sqlx.Open("sqlite3", "./fup.db")
	if err != nil {
		log.Fatal("Something wrong with my db: ", err)
	}
	defer db.Close()
	tx, _ := db.Begin()
	defer tx.Commit()

	cTime := time.Now()
	sql := `
	INSERT INTO uploads (filename, uploadDate) VALUES (?,?);
	`
	if !f.IsDir() {
		db.Exec(sql, f.Name(), cTime)
	}

	return nil
}

// rescanHandler allows for online rescanning of the downloads directory
func rescanHandler(w http.ResponseWriter, r *http.Request) {
	filepath.Walk("./downloads", visitFile)
	http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
}

// handles uploads, copies the file to the filessystem and afterwards
// writes the filename and the upload timestamp to a DB.
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

// downloadHandler blabla for godoc
// The 'customer' wanted a way to stylize the downloads-view,
// Golangs http.FileServer doesn't provide that feature.
// So that's what this function does, it renders a page with a
// list of the uploaded files from the DB.
func downloadHandler(w http.ResponseWriter, r *http.Request) {
	db, err := sqlx.Open("sqlite3", "./fup.db")
	if err != nil {
		log.Fatal("Something wrong with my db: ", err)
	}
	defer db.Close()

	files := []download{}
	err = db.Select(&files, "SELECT id,filename,uploadDate FROM uploads;")
	if err != nil {
		log.Println("Something wrong with the db: ", err)
	}

	dp := &downloadPage{Title: "File Downloads", Downloads: files}
	t, _ := template.ParseFiles("assets/download.html")
	t.Execute(w, dp)
}

// simple function to render the index.html page
func doRest(w http.ResponseWriter, r *http.Request) {
	p := &indexPage{Title: "File UPload"}
	t, _ := template.ParseFiles("assets/index.html")
	t.Execute(w, p)
}
