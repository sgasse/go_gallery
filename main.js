var galleryView = document.querySelector('.flex-container');
var scrollFactor = 15;

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
        galleryView.classList.add('notransition');
        galleryView.style.top = (newTop - elemHeight) + 'px';
        galleryView.offsetHeight; // trigger reflow for CSS changes to take effect
        galleryView.classList.remove('notransition');
        updateItems(-3);
    } else if (newBottom > 0.0) {
        galleryView.classList.add('notransition');
        galleryView.style.top = (newTop + elemHeight) + 'px';
        galleryView.offsetHeight; // trigger reflow for CSS changes to take effect
        galleryView.classList.remove('notransition');
        updateItems(3);
    } else {
        galleryView.style.top = newTop + 'px';
    }

    var otop = parseFloat(gStyle.getPropertyValue('top'), 10);
    var obottom = parseFloat(gStyle.getPropertyValue('bottom'), 10);
    if (DEBUG) {
        console.log("Updated - otop: ", otop, ", obottom: ", obottom);
    }
});

function updateItems(delta) {
    var items = document.querySelectorAll('.flex-item');
    for (var i = 0; i < items.length; i++) {
        var newNumber = parseInt(items[i].innerHTML) + delta;
        items[i].innerHTML = newNumber.toString();
    }
}


function moveToTop() {
    galleryView.style.top = 0 + 'px';
}