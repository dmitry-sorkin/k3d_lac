function download(filename, text) {
    var element = document.createElement('a');
    element.setAttribute('href', 'data:text/plain;charset=utf-8,' + encodeURIComponent(text));
    element.setAttribute('download', filename);

    element.style.display = 'none';
    document.body.appendChild(element);

    element.click();

    document.body.removeChild(element);
}

function saveTextAsFile(filename, text) {
    var textFileAsBlob = new Blob([text], { type: 'text/plain' });

    var downloadLink = document.createElement("a");
    downloadLink.download = filename;
    if (window.webkitURL != null) {
        // Chrome allows the link to be clicked without actually adding it to the DOM.
        downloadLink.href = window.webkitURL.createObjectURL(textFileAsBlob);
    } else {
        // Firefox requires the link to be added to the DOM before it can be clicked.
        downloadLink.href = window.URL.createObjectURL(textFileAsBlob);
        downloadLink.onclick = destroyClickedElement;
        downloadLink.style.display = "none";
        document.body.appendChild(downloadLink);
    }

    downloadLink.click();
}

function showError(value) {
    var container = document.getElementById("resultContainer");
    var output = document.createElement("textarea");
    output.id = "gCode";
    // output.name = "gCode";
    output.cols = "80";
    output.rows = "10";
    output.value = value;
    // output.className = "css-class-name"; // set the CSS class
    container.appendChild(output); //appendChild
}

function destroyClickedElement(event) {
    // remove the link from the DOM
    document.body.removeChild(event.target);
}

var formFields = [
    "k3d_la_bedX",
    "k3d_la_bedY",
    "k3d_la_firmwareMarlin",
    "k3d_la_firmwareKlipper",
    "k3d_la_firmwareRRF",
    "k3d_la_delta",
    "k3d_la_g29",
    "k3d_la_travelSpeed",
    "k3d_la_hotendTemperature",
    "k3d_la_bedTemperature",
    "k3d_la_retractLength",
    "k3d_la_retractSpeed",
    "k3d_la_cooling",
    "k3d_la_flow",
    "k3d_la_firstLayerLineWidth",
    "k3d_la_firstLayerSpeed",
    "k3d_la_zOffset",
    "k3d_la_numPerimeters",
    "k3d_la_lineWidth",
    "k3d_la_layerHeight",
    "k3d_la_fastPrintSpeed",
    "k3d_la_slowPrintSpeed",
    "k3d_la_initKFactor",
    "k3d_la_endKFactor",
    "k3d_la_segmentHeight",
    "k3d_la_numSegments"
];

var saveForm = function () {
    for (var elementId of formFields) {
        var element = document.getElementById(elementId);
        if (element) {
            var saveValue = element.value;
            if (elementId == 'k3d_la_delta' || elementId == 'k3d_la_g29') {
                saveValue = element.checked;
            }
            localStorage.setItem(elementId, saveValue);
        }
    }
}

function loadForm() {
    for (var elementId of formFields) {
        let loadValue = localStorage.getItem(elementId);
        if (loadValue === undefined) {
            continue;
        }

        var element = document.getElementById(elementId);
        if (element) {
            if (elementId == 'k3d_la_delta' || elementId == 'k3d_la_g29') {
                if (loadValue == 'true') {
                    element.checked = true;
                } else {
                    element.checked = false;
                }
                
            } else {
                if (loadValue != null) {
                    element.value = loadValue;
                }
            }
            
        }
    }
}

function initForm() {
    for (var elementId of formFields) {
        var element = document.getElementById(elementId);
        element.onchange = saveForm;
    }
    loadForm();
}
