package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"strconv"
	"text/template"
)

var (
	port = flag.Int("port", 3353, "Port to listen on")
)

type restData struct {
	GalleryContent string `json:"GalleryContent"`
	LastRow        bool   `json:"LastRow"`
}

type tplData struct {
	DivContent []string
}

func getDivNums(rowOffset int) tplData {
	dummyContent := make([]string, 15)
	for ind := range dummyContent {
		dummyContent[ind] = strconv.Itoa(ind + 1 + rowOffset*3)
	}

	dm := tplData{DivContent: dummyContent}
	return dm
}

func tmpGetHtmContent(rowOffset int) []byte {
	t, _ := template.ParseFiles("templates/galleryContent.html")

	dm := getDivNums(rowOffset)

	var tplOut bytes.Buffer
	if err := t.Execute(&tplOut, dm); err != nil {
		log.Fatal(err)
	}

	return tplOut.Bytes()
}

func galleryHandler(w http.ResponseWriter, r *http.Request) {
	t, _ := template.ParseFiles("templates/gallery.html")
	dm := getDivNums(0)

	t.Execute(w, dm)
}

type clientCall struct {
	FirstRow int `json:"FirstRow"`
}

func restGalleryHandler(w http.ResponseWriter, r *http.Request) {
	var cCall clientCall
	err := json.NewDecoder(r.Body).Decode(&cCall)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	htmlBody := tmpGetHtmContent(cCall.FirstRow)
	rData := restData{GalleryContent: string(htmlBody), LastRow: false}

	jBody, err := json.MarshalIndent(rData, "", "    ")
	if err != nil {
		log.Fatal(err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(jBody)
}

func main() {
	flag.Parse()

	mux := http.NewServeMux()
	fs := http.FileServer(http.Dir("assets"))

	mux.HandleFunc("/gallery", galleryHandler)
	mux.HandleFunc("/restGallery", restGalleryHandler)
	mux.Handle("/assets/", http.StripPrefix("/assets/", fs))
	http.ListenAndServe(":"+strconv.Itoa(*port), mux)
}
