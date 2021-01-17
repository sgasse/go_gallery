var last_scroll = 0;
var firstRow = 0;

function zoomImg(img) {
    console.log("Zooming in on ", img);
    var focus = document.querySelector('.focus');
    focus.style.backgroundImage = "url('" + img + "')";
    focus.style.zIndex = "1";
    focus.style.opacity = "1.0";
}

function hideZoom() {
    console.log("Hiding zoom")
    var focus = document.querySelector('.focus');
    focus.style.opacity = "0.0";
    setTimeout(() => { focus.style.zIndex = "-1"; }, 300);
}

const fetchImgData = (firstRow) =>
    fetch('http://127.0.0.1:3353/restGallery', {
        method: 'POST',
        body: JSON.stringify({ 'FirstRow': firstRow }),
        headers: {
            'Content-Type': 'application/json'
        }
    }).then(response => response.json());


function handleSrvResp(resp) {
    var galleryView = document.querySelector('.flex-container');
    galleryView.innerHTML = resp.GalleryContent;
    console.log('Image data updated for top row ', firstRow)
}

function imgHeight() {
    var elem = document.querySelector('.flex-item');
    var elemHeight = elem.offsetHeight;
    elemHeight += parseInt(window.getComputedStyle(elem).getPropertyValue('margin-top'))
    elemHeight += parseInt(window.getComputedStyle(elem).getPropertyValue('margin-bottom'))
    return elemHeight;
}

document.addEventListener('scroll', function(e) {
    var elemHeight = imgHeight();
    if (Math.abs(last_scroll - window.scrollY) > 2 * elemHeight) {
        last_scroll = window.scrollY;
        firstRow = Math.floor(window.scrollY / elemHeight);
        fetchImgData(firstRow).then(handleSrvResp);
    }
});