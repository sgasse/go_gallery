package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"
)

var (
	port     = flag.Int("port", 3353, "Port to listen on")
	dir      = flag.String("dir", "./", "Directory for which to look for images")
	rows     = flag.Int("rows", 3, "Number of rows in gallery view")
	cols     = flag.Int("cols", 3, "Number of columns in gallery view")
	prefetch = flag.Int("prefetch", 3, "Number of rows to prefetch above and below")
)

// imgs contains the paths to image files recursively found in `dir`.
var imgs []string

// restData is the JSON-formatted response send to the JS callback.
type restData struct {
	GalleryContent string `json:"GalleryContent"`
}

// tplData contains the fields used in templating the HTML views.
type tplData struct {
	DivContent []string
}

// jsReq is the JSON-formatted request data from the JS callback.
type jsReq struct {
	FirstRow int `json:"FirstRow"`
}

func parseImgs(inPath string) []string {
	var imgPaths []string

	selectImgs := func(path string, info os.FileInfo, err error) error {
		if err == nil {
			if strings.HasSuffix(path, ".jpg") || strings.HasSuffix(path, ".jpeg") || strings.HasSuffix(path, ".png") {
				imgPaths = append(imgPaths, "imgs/"+strings.TrimPrefix(path, *dir))
			}
		}
		return nil
	}

	err := filepath.Walk(inPath, selectImgs)
	if err != nil {
		log.Fatal(err)
	}
	return imgPaths
}

// maskImgView creates a img path array with all images not in view or the prefetch areas masked out.
func maskImgView(firstRow int) tplData {
	startIdx := 0
	endIdx := (firstRow + *rows + *prefetch) * *cols

	if firstRow >= *prefetch {
		startIdx = (firstRow - *prefetch) * *cols
	}

	if endIdx > len(imgs) {
		endIdx = len(imgs)
	}

	maskedImgs := make([]string, startIdx)
	maskedImgs = append(maskedImgs, imgs[startIdx:endIdx]...)
	maskedImgs = append(maskedImgs, make([]string, len(imgs)-endIdx)...)

	return tplData{maskedImgs}
}

func getGalleryHTML(rowOffset int) []byte {
	t, _ := template.ParseFiles("templates/galleryContent.html")

	dm := maskImgView(rowOffset)

	var tplOut bytes.Buffer
	if err := t.Execute(&tplOut, dm); err != nil {
		log.Fatal(err)
	}

	return tplOut.Bytes()
}

func restGalleryHandler(w http.ResponseWriter, r *http.Request) {
	var req jsReq
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	htmlBody := getGalleryHTML(req.FirstRow)
	rData := restData{GalleryContent: string(htmlBody)}

	jBody, err := json.MarshalIndent(rData, "", "    ")
	if err != nil {
		log.Fatal(err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(jBody)
}

func galleryHandler(w http.ResponseWriter, r *http.Request) {
	t, _ := template.ParseFiles("templates/gallery.html")
	dm := maskImgView(0)
	t.Execute(w, dm)
}

func main() {
	flag.Parse()

	imgs = parseImgs(*dir)

	mux := http.NewServeMux()

	fs := http.FileServer(http.Dir("assets"))
	imgFs := http.FileServer(http.Dir(*dir))

	mux.HandleFunc("/gallery", galleryHandler)
	mux.HandleFunc("/restGallery", restGalleryHandler)
	mux.Handle("/assets/", http.StripPrefix("/assets/", fs))
	mux.Handle("/imgs/", http.StripPrefix("/imgs/", imgFs))
	http.ListenAndServe("127.0.0.1:"+strconv.Itoa(*port), mux)
}
