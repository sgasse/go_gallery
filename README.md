# Minimalist's Gallery

This repository features a photo gallery to scroll through images found
recursively in a directory. It is based on a Go server backend and an
HTML/CSS/JS frontend in the browser.

The server backend templates a HTML page with as many entries as it
finds pictures. However only the images in view plus some pre-fetch
margin are put into the HTML document. This way, the browser is never
overwhelmed with the amount of images to hold in the cache. You can
zoom in/out on images by clicking on them.

You can build the project with:
```
go build -o migallery
```

Run it like this:
```
./migallery --dir "my/path/to/look/for/images"
```

The gallery will be available by default at
http://127.0.0.1:3353/gallery
but you can customize the port with the flag `--port`.
