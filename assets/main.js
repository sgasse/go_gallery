var scrollFactor = 1;
var galleryView = document.querySelector('.flex-container');

var firstRow = 0;

var DEBUG = false;

galleryView.addEventListener('wheel', function(e) {
    var gStyle = window.getComputedStyle(galleryView);

    var otop = parseFloat(gStyle.getPropertyValue('top'), 10);
    var obottom = parseFloat(gStyle.getPropertyValue('bottom'), 10);
    if (DEBUG) {
        console.log("Old - otop: ", otop, ", obottom: ", obottom);
    }

    var newTop = otop + scrollFactor * e.deltaY;
    var newBottom = obottom - scrollFactor * e.deltaY; // substract from bottom
    if (DEBUG) {
        console.log("New top: ", newTop, ", new bottom: ", newBottom);
    }

    var elemHeight = document.querySelector('.flex-item').clientHeight;

    if (newTop > 0.0) {
        // scroll would move out of grid on the top
        if (firstRow > 0) {
            // current top row is not the first row
            moveViewTo((newTop - elemHeight) + 'px');
            firstRow -= 1;
            updatedData(firstRow).then(handleSrvResp);
        } else {
            // current top row is first row
            moveViewTo(0 + 'px');
        }
    } else if (newBottom > 0.0) {
        // scroll would move out of grid on the bottom
        moveViewTo((newTop + elemHeight) + 'px');
        firstRow += 1;
        updatedData(firstRow).then(handleSrvResp);
    } else {
        galleryView.style.top = newTop + 'px';
    }

    var otop = parseFloat(gStyle.getPropertyValue('top'), 10);
    var obottom = parseFloat(gStyle.getPropertyValue('bottom'), 10);
    if (DEBUG) {
        console.log("Updated - otop: ", otop, ", obottom: ", obottom);
    }
});

function moveViewTo(topPos) {
    galleryView.classList.add('notransition');
    galleryView.style.top = topPos;
    galleryView.offsetHeight; // trigger reflow for CSS changes to take effect
    galleryView.classList.remove('notransition');
}

function handleSrvResp(resp) {
    galleryView.innerHTML = resp.GalleryContent
}

// Arrow-function definition of updated data
// Test out with `updatedData().then(console.log)
const updatedData = (firstRow) =>
    fetch('http://127.0.0.1:3353/restGallery', {
        method: 'POST',
        body: JSON.stringify({ 'FirstRow': firstRow }),
        headers: {
            'Content-Type': 'application/json'
        }
    }).then(response => response.json());


// --- DEBUG functions ---

function moveToTop() {
    galleryView.style.top = 0 + 'px';
}

function updateItems() {
    var items = document.querySelectorAll('.flex-item');
    for (var i = 0; i < items.length; i++) {
        //var newNumber = parseInt(items[i].innerHTML) + delta;
        items[i].innerHTML = i.toString();
    }
}