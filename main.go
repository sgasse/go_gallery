package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"log"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/h2non/bimg"
)

var (
	port      = flag.Int("port", 3353, "Port to listen on")
	dir       = flag.String("dir", "./", "Directory for which to look for images")
	rows      = flag.Int("rows", 3, "Number of rows in gallery view")
	cols      = flag.Int("cols", 3, "Number of columns in gallery view")
	prefetch  = flag.Int("prefetch", 3, "Number of rows to prefetch above and below")
	randomize = flag.Bool("randomize", false, "Random shuffle images")
)

type convTask struct {
	nameTarget *string
	file       *string
}

// paths to image files recursively found in `dir`, including absolute, relative and thumbnail paths.
var absImgs, imgs, thumbs []string

var thumbDir string
var convChan chan convTask

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

func createPreview(file, outdir string) (outfile string, err error) {
	buffer, err := bimg.Read(file)
	if err != nil {
		return
	}

	newImage, err := bimg.NewImage(buffer).Resize(800, 600)
	if err != nil {
		return
	}

	outfile = filepath.Join(outdir, filepath.Base(file))

	bimg.Write(outfile, newImage)
	return
}

func parseImgs(inPath string) (absPaths, imgPaths []string) {
	selectImgs := func(path string, info os.FileInfo, err error) error {
		if err == nil {
			if strings.HasSuffix(path, ".jpg") || strings.HasSuffix(path, ".jpeg") || strings.HasSuffix(path, ".png") {
				absPaths = append(absPaths, path)
				imgPaths = append(imgPaths, "imgs/"+strings.TrimPrefix(path, *dir))
			}
		}
		return nil
	}

	err := filepath.Walk(inPath, selectImgs)
	if err != nil {
		log.Fatal(err)
	}
	return absPaths, imgPaths
}

func getMaskInds(firstRow, prefetch int) (startIdx, endIdx int) {
	// startIdx initialized as zero
	endIdx = (firstRow + *rows + prefetch) * *cols

	if firstRow >= prefetch {
		startIdx = (firstRow - prefetch) * *cols
	}

	if endIdx > len(imgs) {
		endIdx = len(imgs)
	}
	return
}

// maskImgView creates a img path array with all images not in view or the prefetch areas masked out.
func maskImgView(firstRow int) tplData {
	startIdx, endIdx := getMaskInds(firstRow, *prefetch)

	maskedImgs := make([]string, startIdx)
	maskedImgs = append(maskedImgs, imgs[startIdx:endIdx]...)
	maskedImgs = append(maskedImgs, make([]string, len(imgs)-endIdx)...)

	return tplData{maskedImgs}
}

func getGalleryHTML(rowOffset int) []byte {
	t, _ := template.ParseFiles("templates/galleryContent.html")

	data := maskImgView(rowOffset)

	var tplOut bytes.Buffer
	if err := t.Execute(&tplOut, data); err != nil {
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

	go findToThumbnail(req.FirstRow, convChan)
	htmlBody := getGalleryHTML(req.FirstRow)
	rData := restData{GalleryContent: string(htmlBody)}

	jBody, err := json.MarshalIndent(rData, "", "    ")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Fatal(err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(jBody)
}

func galleryHandler(w http.ResponseWriter, r *http.Request) {
	t, _ := template.ParseFiles("templates/gallery.html")
	data := maskImgView(0)
	t.Execute(w, data)
}

func randomizeData(arr1, arr2 []string) ([]string, []string) {
	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(arr1), func(i, j int) {
		arr1[i], arr1[j] = arr1[j], arr1[i]
		arr2[i], arr2[j] = arr2[j], arr2[i]
	})
	return arr1, arr2
}

func findToThumbnail(firstRow int, toConv chan<- convTask) {
	startIdx, endIdx := getMaskInds(firstRow, *prefetch+2)
	for i := startIdx; i < endIdx; i++ {
		if thumbs[i] == "" {
			toConv <- convTask{&thumbs[i], &absImgs[i]}
		}
	}
}

func thumbnailImgs(toConv <-chan convTask) {
	for {
		task := <-toConv
		outfile, err := createPreview(*task.file, thumbDir)
		if err != nil {
			log.Fatal(err)
		}

		*task.nameTarget = outfile
	}
}

func main() {
	flag.Parse()

	absImgs, imgs = parseImgs(*dir)
	if *randomize {
		absImgs, imgs = randomizeData(absImgs, imgs)
	}
	thumbs = make([]string, len(absImgs))

	thumbDir = "/tmp/goGalleryThumbs"
	err := os.Mkdir(thumbDir, 0755)
	if err != nil {
		log.Fatal(err)
	}

	convChan = make(chan convTask, 8)
	go findToThumbnail(0, convChan)
	go thumbnailImgs(convChan)

	mux := http.NewServeMux()

	fs := http.FileServer(http.Dir("assets"))
	imgFs := http.FileServer(http.Dir(*dir))

	mux.HandleFunc("/gallery", galleryHandler)
	mux.HandleFunc("/restGallery", restGalleryHandler)
	mux.Handle("/assets/", http.StripPrefix("/assets/", fs))
	mux.Handle("/imgs/", http.StripPrefix("/imgs/", imgFs))
	http.ListenAndServe("127.0.0.1:"+strconv.Itoa(*port), mux)
}
