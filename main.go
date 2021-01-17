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
	port = flag.Int("port", 3353, "Port to listen on")
	dir  = flag.String("dir", "./", "Directory for which to look for images")
	rows = flag.Int("rows", 3, "Number of rows in gallery view")
	cols = flag.Int("cols", 3, "Number of columns in gallery view")
)

var imgs []string

type restData struct {
	GalleryContent string `json:"GalleryContent"`
	LastRow        bool   `json:"LastRow"`
}

type tplData struct {
	DivContent []string
}

func getImgData(firstRow int) tplData {
	startIdx := *cols * firstRow
	endIdx := startIdx + *cols*(*rows+2)

	if endIdx < len(imgs) {
		return tplData{imgs[startIdx:endIdx]}
	}

	imgPaths := imgs[startIdx:]
	remainder := make([]string, len(imgs)-endIdx)
	imgPaths = append(imgPaths, remainder...)

	return tplData{imgPaths}
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

	dm := getImgData(rowOffset)

	var tplOut bytes.Buffer
	if err := t.Execute(&tplOut, dm); err != nil {
		log.Fatal(err)
	}

	return tplOut.Bytes()
}

func galleryHandler(w http.ResponseWriter, r *http.Request) {
	t, _ := template.ParseFiles("templates/gallery.html")
	dm := getImgData(0)
	t.Execute(w, dm)
}

type clientCall struct {
	FirstRow int `json:"FirstRow"`
}

func getImages(inPath string) []string {
	var imgPaths []string
	appendFile := func(path string, info os.FileInfo, err error) error {
		if err == nil {
			if strings.HasSuffix(path, ".jpg") || strings.HasSuffix(path, ".jpeg") || strings.HasSuffix(path, ".png") {
				imgPaths = append(imgPaths, "imgs/"+strings.TrimPrefix(path, *dir))
			}
		}
		return nil
	}
	err := filepath.Walk(inPath, appendFile)
	if err != nil {
		log.Fatal(err)
	}
	return imgPaths
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

	imgs = getImages(*dir)
	log.Println("Length of imgs: ", len(imgs))
	remainder := make([]string, *cols-(len(imgs)%*cols))
	imgs = append(imgs, remainder...)
	log.Println("Length of imgs: ", len(imgs))

	mux := http.NewServeMux()
	fs := http.FileServer(http.Dir("assets"))

	imgFs := http.FileServer(http.Dir(*dir))

	mux.HandleFunc("/gallery", galleryHandler)
	mux.HandleFunc("/restGallery", restGalleryHandler)
	mux.Handle("/assets/", http.StripPrefix("/assets/", fs))
	mux.Handle("/imgs/", http.StripPrefix("/imgs/", imgFs))
	http.ListenAndServe(":"+strconv.Itoa(*port), mux)
}
