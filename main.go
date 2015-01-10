package main

import (
	//"fmt"
	//"io/ioutil"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
)

type page struct {
	Title string
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
		log.Println("Kabutt: ", err)
	}
	defer file.Close()

	out, err := os.Create("./downloads/" + header.Filename)
	defer out.Close()
	if err != nil {
		log.Println("Kabutt: ", err)
	}
	_, err = io.Copy(out, file)
	if err != nil {
		log.Println("Kabutt: ", err)
	}

	http.Redirect(w, r, "/downloads", http.StatusFound)

}
func doRest(w http.ResponseWriter, r *http.Request) {
	p := &page{Title: "File UPload"}
	t, _ := template.ParseFiles("index.html")
	t.Execute(w, p)
}
