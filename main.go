package main

import (
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
)

const (
	pageURL = "do.krzbff.de"
)

type page struct {
	Title string
	Path  string
	Body  []byte
}

func init() {
	os.Mkdir("./downloads", 0755)
}

func main() {
	http.Handle("/downloads/", http.StripPrefix("/downloads/", http.FileServer(http.Dir("./downloads"))))
	http.HandleFunc("/upload", uploadHandler)
	http.HandleFunc("/", doRest)
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func uploadHandler(w http.ResponseWriter, r *http.Request) {
	file, header, err := r.FormFile("file")
	if err != nil {
		log.Println("Error while uploading: ", err)
	}
	defer file.Close()

	out, err := os.Create("./downloads/" + header.Filename)
	defer out.Close()
	if err != nil {
		log.Println("Error while creating file: ", err)
	}
	_, err = io.Copy(out, file)
	if err != nil {
		log.Println("Error while writing to file: ", err)
	}

	http.Redirect(w, r, "/downloads", http.StatusFound)

}
func doRest(w http.ResponseWriter, r *http.Request) {
	p := &page{Title: "File UPload", Path: pageURL}
	t, _ := template.ParseFiles("index.html")
	t.Execute(w, p)
}
