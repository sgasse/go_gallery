package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"text/template"
	"time"

	"github.com/h2non/bimg"
)

var (
	port       = flag.Int("port", 3353, "Port to listen on")
	dir        = flag.String("dir", "./", "Directory for which to look for images")
	rows       = flag.Int("rows", 3, "Number of rows in gallery view")
	cols       = flag.Int("cols", 3, "Number of columns in gallery view")
	randomize  = flag.Bool("randomize", false, "Random shuffle images")
	numWorkers = flag.Int("numWorkers", 4, "Number of resize worker routines")
)

type galleryImg struct {
	AbsPath    string
	ServerPath string
	ThumbPath  string
}

var imgs []galleryImg

var thumbDir string
var convChan chan *galleryImg
var wg sync.WaitGroup

// restData is the JSON-formatted response send to the JS callback.
type restData struct {
	GalleryContent string `json:"GalleryContent"`
	DebounceRows   int    `json:"DebounceRows"`
}

// tplData contains the fields used in templating the HTML views.
type tplData struct {
	Images          []galleryImg
	ItemWidth       string
	ItemHeight      string
	ItemMarginHoriz string
	ItemMarginVert  string
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

	newImage, err := bimg.NewImage(buffer).Resize(0, 800)
	if err != nil {
		return
	}

	outfile = filepath.Join(outdir, filepath.Base(file))

	bimg.Write(outfile, newImage)
	return
}

func randomizeData(data []galleryImg) []galleryImg {
	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(data), func(i, j int) { data[i], data[j] = data[j], data[i] })
	return data
}

func parseImgs(inPath string, randomize bool) (parsedImgs []galleryImg) {
	selectImgs := func(path string, info os.FileInfo, err error) error {
		if err == nil {
			if strings.HasSuffix(path, ".jpg") || strings.HasSuffix(path, ".jpeg") || strings.HasSuffix(path, ".png") {
				im := galleryImg{AbsPath: path, ServerPath: "imgs/" + strings.TrimPrefix(path, *dir)}
				parsedImgs = append(parsedImgs, im)
			}
		}
		return nil
	}

	err := filepath.Walk(inPath, selectImgs)
	if err != nil {
		log.Fatal(err)
	}

	if randomize {
		parsedImgs = randomizeData(parsedImgs)
	}

	return parsedImgs
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

func getHwmStyle() (w, h, mv, mh string) {
	w = fmt.Sprintf("%.2f", 99.0/float32(*cols))
	h = fmt.Sprintf("%.2f", 99.0/float32(*rows))
	mv = fmt.Sprintf("%.2f", 1.0/(2*float32(*rows)))
	mh = fmt.Sprintf("%.2f", 1.0/(2*float32(*cols)))
	return w, h, mv, mh
}

func maskImgView(firstRow int) tplData {
	// Prefetch one complete screen above and below
	startIdx, endIdx := getMaskInds(firstRow, *rows)

	maskedImgs := make([]galleryImg, startIdx)
	maskedImgs = append(maskedImgs, imgs[startIdx:endIdx]...)
	maskedImgs = append(maskedImgs, make([]galleryImg, len(imgs)-endIdx)...)

	w, h, mv, mh := getHwmStyle()

	return tplData{Images: maskedImgs, ItemWidth: w, ItemHeight: h,
		ItemMarginVert: mv, ItemMarginHoriz: mh}

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

	findToThumbnail(req.FirstRow, convChan)
	wg.Wait()
	htmlBody := getGalleryHTML(req.FirstRow)
	debounceR := int(math.Ceil(float64(*rows) / 2.0))
	rData := restData{GalleryContent: string(htmlBody), DebounceRows: debounceR}

	jBody, err := json.MarshalIndent(rData, "", "    ")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Fatal(err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(jBody)
}

func galleryHandler(w http.ResponseWriter, r *http.Request) {
	t, parseErr := template.ParseFiles("templates/gallery.html")
	if parseErr != nil {
		log.Fatal(parseErr)
	}
	content := string(getGalleryHTML(0))
	t.Execute(w, content)
}

func findToThumbnail(firstRow int, toConv chan<- *galleryImg) {
	// Pre-thumbnail two complete screens above and below
	startIdx, endIdx := getMaskInds(firstRow, 2*(*rows))
	for i := startIdx; i < endIdx; i++ {
		if imgs[i].ThumbPath == "" {
			wg.Add(1)
			toConv <- &imgs[i]
		}
	}
}

func thumbnailImgs(toConv <-chan *galleryImg) {
	for {
		im := <-toConv
		outfile, err := createPreview(im.AbsPath, thumbDir)
		if err != nil {
			log.Fatal(err)
		}

		im.ThumbPath = "thumbs/" + filepath.Base(outfile)
		wg.Done()
	}
}

func shutdown(sigChan chan os.Signal, thumbsDir string) {
	<-sigChan
	log.Println("Received SIGINT/SIGTERM, removing thumbnail directory ", thumbsDir)
	os.RemoveAll(thumbsDir)
	os.Exit(0)
}

func main() {
	flag.Parse()

	os.Setenv("VIPS_WARNING", "0")

	// catch shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	imgs = parseImgs(*dir, *randomize)

	thumbDir = "/tmp/goGalleryThumbs"
	err := os.MkdirAll(thumbDir, 0755)
	if err != nil {
		log.Fatal(err)
	}
	go shutdown(sigChan, thumbDir)

	chanSize := 5 * (*rows) * (*cols)
	convChan = make(chan *galleryImg, chanSize)
	go findToThumbnail(0, convChan)
	for i := 0; i < *numWorkers; i++ {
		go thumbnailImgs(convChan)
	}

	mux := http.NewServeMux()

	fs := http.FileServer(http.Dir("assets"))
	fullResFs := http.FileServer(http.Dir(*dir))
	thumbsFs := http.FileServer(http.Dir(thumbDir))

	mux.HandleFunc("/gallery", galleryHandler)
	mux.HandleFunc("/restGallery", restGalleryHandler)
	mux.Handle("/assets/", http.StripPrefix("/assets/", fs))
	mux.Handle("/imgs/", http.StripPrefix("/imgs/", fullResFs))
	mux.Handle("/thumbs/", http.StripPrefix("/thumbs/", thumbsFs))
	http.ListenAndServe("127.0.0.1:"+strconv.Itoa(*port), mux)
}
