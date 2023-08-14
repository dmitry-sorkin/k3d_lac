const calibrator_version = 'v0.1';
window.calibrator_version = calibrator_version;
var savedSegmentsInfo = null;

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

var currentStream = null;
var currentWriter = null;
const encoder = new TextEncoder();

function beginSaveFile(filename) {
	if (currentStream != null) {
		currentStream.abort();
	}
	currentStream = streamSaver.createWriteStream(filename);
	currentWriter = currentStream.getWriter();
}

function writeToFile(str) {
	if (currentWriter != null) {
		currentWriter.write(encoder.encode(str));
	}
}

function finishFile() {
	if (currentWriter != null) {
		currentWriter.ready.then(() => {
			currentWriter.close();
			currentWriter = null;
		})
		.catch((err)=>{
			showError(err);
		});
	}
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
	// Параметры принтера
    "k3d_smc_bedX",
    "k3d_smc_bedY",
    "k3d_smc_firmwareMarlin",
    "k3d_smc_firmwareKlipper",
    "k3d_smc_firmwareRRF",
    "k3d_smc_delta",
    "k3d_smc_bedProbe",
	// Параметры филамента
	"k3d_smc_hotendTemperature",
	"k3d_smc_bedTemperature",
	"k3d_smc_cooling",
	"k3d_smc_flow",
	"k3d_smc_la",
	// Параметры первого слоя
	"k3d_smc_firstLayerSpeed",
	"k3d_smc_firstLayerLineWidth",
	"k3d_smc_zOffset",
	// Параметры модели
	"k3d_smc_lineWidth",
	"k3d_smc_layerHeight",
	"k3d_smc_printSpeed",
	"k3d_smc_slowAcceleration"
];
var segmentFields = [
	"k3d_smc_startAcceleration",
	"k3d_smc_endAcceleration",
	"k3d_smc_numSegments"
];
var segmentKeys = [
	"init_acc",
	"end_acc",
	"num_segments"
];

var saveForm = function () {
    for (var elementId of formFields) {
        var element = document.getElementById(elementId);
        if (element) {
            var saveValue = element.value;
            if (elementId == 'k3d_smc_delta' || elementId == 'k3d_smc_g29') {
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
            if (elementId == 'k3d_smc_delta' || elementId == 'k3d_smc_g29') {
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
    loadForm();
	for (var elementId of formFields) {
        var element = document.getElementById(elementId);
        element.addEventListener('change', function(e) {
			saveForm();
			
			var el = e.target;
			var id = el.id;
			
			if (segmentFields.indexOf(id) != -1) {
				checkSegments();
			} else {
				checkGo();
			}
		});
    }
	for (var elementId of segmentFields) {
		var element = document.getElementById(elementId);
		element.addEventListener('focusin', function(e) {
			checkSegments();
		});
		element.addEventListener('focusout', function(e) {
			if (e.relatedTarget == undefined || segmentFields.indexOf(e.relatedTarget.id) == -1) {
				checkGo();
			}
		});
	}
}

function initLang(key) {
	var values = window.lang.values;
	switch (key) {
		case 'en':

			break;
		case 'ru':
			// Заголовок

			values['header.title'] = 'K3D калибровщик максимальных ускорений для IS';
			values['header.language'] = 'Язык: ';
			values['header.instruction'] = 'Инструкция по использованию';

			// Заголовок таблицы
			
			values['table.header.parameter'] = 'Параметр';
			values['table.header.value'] = 'Значение';
			values['table.header.description'] = 'Описание';
			
			// Параметры принтера

			values['table.bed_size_x.title'] = 'Размер стола по X';
			values['table.bed_size_x.description'] = '[мм] Для декартовых принтеров - максимальная координата по оси X<br>Для дельта-принтеров - <b>диаметр стола</b>';
			
			values['table.bed_size_y.title'] = 'Размер стола по Y';
			values['table.bed_size_y.description'] = '[мм] Для декартовых принтеров - максимальная координата по оси Y<br>Для дельта-принтеров - <b>диаметр стола</b>';
			
			values['table.firmware.title'] = 'Прошивка';
			values['table.firmware.description'] = 'Прошивка, установленная на вашем принтере';
			
			values['table.delta.title'] = 'Начало координат в центре стола';
			values['table.delta.description'] = 'Для декартовых принтеров должно быть выключено, для дельт включено';
			
			values['table.bed_probe.title'] = 'Автокалибровка стола';
			values['table.bed_probe.description'] = 'Надо ли делать автокалибровку стола перед печатью (G29)? Если у вас нет датчика автокалибровки, то оставляйте выключенным';
			
			// Параметры филамента

			values['table.hotend_temp.title'] = 'Температура хотэнда';
			values['table.hotend_temp.description'] = '[°C] До прогрева стола хотэнд будет нагрет до 150 градусов. После полного нагрева стола хотэнд догреется до указанной температуры';
			
			values['table.bed_temp.title'] = 'Температура стола';
			values['table.bed_temp.description'] = '[°C] Температура, до которой нагреть стол перед печатью. Стол будет нагрет до выполнения парковки и автокалибровки стола';
			
			values['table.cooling.title'] = 'Скорость вентилятора';
			values['table.cooling.description'] = '[%] Обороты вентилятора в процентах. Для того, чтобы температура хотэнда резко не упала при включении вентилятора, на 1 слое обдув будет выключен. На 2-4 слоях скорость вращения вентиляторов будет ступенчато увелиичиваться до указанного значения';
			
			values['table.flow.title'] = 'Поток';
			values['table.flow.description'] = '[%] Поток в процентах. Нужен для компенсации пере- или недоэкструзии';

			values['table.la.title'] = 'k-фактор LA/PA';
			values['table.la.description'] = '[с] Введите сюда ваше значение для Linear/Pressure Advance. Если вы не пользуетесь Linear/Pressure Advance, то оставьте значение нулевым';
			
			// Параметры первого слоя

			values['table.first_line_width.title'] = 'Ширина линии первого слоя';
			values['table.first_line_width.description'] = '[мм] Ширина линий, с которой будет напечатана подложка под моделью. В общем случае рекомендуется выставить 150% от диаметра сопла';
			
			values['table.first_print_speed.title'] = 'Скорость печати первого слоя';
			values['table.first_print_speed.description'] = '[мм/с] Скорость, с которой будет напечатана подложка';
			
			values['table.z_offset.title'] = 'Z-offset';
			values['table.z_offset.description'] = '[мм] Смещение всей модели по вертикали. Нужно чтобы компенсировать слишком тонкую/толстую калибровку первого слоя. В общем случае оставьте ноль';
			
			// Параметры модели

			values['table.line_width.title'] = 'Ширина линии';
			values['table.line_width.description'] = '[мм] Ширина линий, с которой будут напечатаны башенки. В общем случае рекомендуется выставить равной диаметру сопла';
			
			values['table.layer_height.title'] = 'Толщина слоя';
			values['table.layer_height.description'] = '[мм] Толщина слоёв всей модели. В общем случае 50% от ширины линии';
			
			values['table.start_gcode.title'] = 'Начальный G-код';
			values['table.start_gcode.description'] = 'Код, выполняемый перед печатью теста. Менять на свой страх и риск! Список возможных плейсхолдеров:<br><b>$BEDTEMP</b> - температура стола<br><b>$HOTTEMP</b> - температура хотэнда<br><b>$G29</b> - команда на снятие карты высот стола<br><b>$FLOW</b> - поток';
			
			values['table.end_gcode.title'] = 'Конечный G-код';
			values['table.end_gcode.description'] = 'Код, выполняемый после печати теста. Менять на свой страх и риск!';
			
			// Параметры калибровки

			values['table.num_segments.title'] = 'Количество сегментов';
			values['table.num_segments.description'] = 'Количество сегментов башенки. В течение сегмента коэффициент LA остаётся неизменным. Сегменты визуально разделены для упрощения анализа модели';

			values['table.start_acceleration.title'] = 'Начальное значение ускорения';
			values['table.start_acceleration.description'] = '[мм/с^2] Начальное значение ускорения для калибровки';

			values['table.end_acceleration.title'] = 'Конечное значение ускорения';
			values['table.end_acceleration.description'] = '[мм/с^2] Конечное значение ускорения для калибровки';

			// Генератор
			
			values['generator.generate_and_download'] = 'Генерировать и скачать';		
			values['generator.generate_button_loading'] = 'Генератор загружается...';		
			values['generator.segment'] = '; Сегмент %d: Ускорение: %s\n';
			values['generator.reset_to_default'] = 'Сбросить настройки';

			// Футер
			
			values['navbar.back'] = ' Назад ';
			values['navbar.site'] = 'Сайт';
			
			// Ошибки параметров принтера

			values['error.bed_size_x.format'] = 'Размер оси Х - ошибка формата';
			values['error.bed_size_x.value'] = 'Размер стола по X указан неверно (меньше 100 или больше 1000 мм)';

			values['error.bed_size_y.format'] = 'Размер оси Y - ошибка формата';
			values['error.bed_size_y.value'] = 'Размер стола по Y указан неверно (меньше 100 или больше 1000 мм)';

			values['error.firmware.not_set'] = 'Ошибка формата: не выбрана прошивка';

			// Ошибки параметров филамента

			values['error.hotend_temp.format'] = 'Температура хотэнда - ошибка формата';
			values['error.hotend_temp.value'] = 'Температура хотэнда указана неверно (меньше 150 или больше 350 градусов)';

			values['error.bed_temp.format'] = 'Температура стола - ошибка формата';
			values['error.bed_temp.too_high'] = 'Температура стола слишком высокая';

			values['error.cooling.format'] = 'Скорость вентилятора - ошибка формата';

			values['error.flow.format'] = 'Поток - ошибка формата';
			values['error.flow.value'] = 'Поток указан неверно (меньше 50 или больше 150%)';

			values['error.la.format'] = 'k-фактор LA/PA - ошибка формата';
			values['error.la.value'] = 'k-фактор LA/PA указан неверно (меньше 0.0 или больше 2.0)'

			// Ошибки параметров первого слоя

			values['error.first_line_width.format'] = 'Ширина линии первого слоя - ошибка формата';
			values['error.first_line_width.value'] = 'Неправильная ширина линии первого слоя (меньше 0.1 или больше 2.0 мм)';

			values['error.first_layer_speed.format'] = 'Скорость печати первого слоя - ошибка формата';
			values['error.first_layer_speed.slow_or_fast'] = 'Скорость печати первого слоя неправильная (меньше 10 или больше 1000 мм/с)';

			values['error.z_offset.format'] = 'Z-offset - ошибка формата';
			values['error.z_offset.value'] = 'Значение оффсета неправильно (меньше -0.5 или больше 0.5 мм)';

			// Ошибки параметров модели

			values['error.line_width.format'] = 'Ширина линии - ошибка формата';
			values['error.line_width.value'] = 'Неправильная ширина линии (меньше 0.1 или больше 2.0 мм)';

			values['error.layer_height.format'] = 'Высота слоя - ошибка формата';
			values['error.layer_height.value'] = 'Толщина слоя неправильная (меньше 0.05 или больше 1.2 мм)';

			values['error.print_speed.format'] = 'Скорость печати - ошибка формата';
			values['error.print_speed.value'] = 'Скорость печати неправильная (меньше 60 или больше 600 мм/с';

			values['error.slow_acceleration.format'] = 'Ускорение для медленных сегментов - ошибка формата';
			values['error.slow_acceleration.value'] = 'Ускорение для медленных сегментов указано неверно (меньше 100 или больше 50000 мм/с^2';

			// Ошибки параметров калибровки

			values['error.start_acceleration.title'] = 'Начальное значение ускорения - ошибка формата';
			values['error.start_acceleration.value'] = 'Начальное значение ускорения указано неверно (меньше 100 или больше 50000 мм/с^2)';

			values['error.end_acceleration.title'] = 'Конечное значение ускорения - ошибка формата';
			values['error.end_acceleration.value'] = 'Конечное значение ускорения указано неверно (меньше 100 или больше 50000 мм/с^2)';

			values['error.num_segments.format'] = 'Количество сегментов - ошибка формата';
			values['error.num_segments.value'] = 'Количество сегментов неправильное (меньше 2 или больше 100)';

			break;
	}
	
	document.title = window.lang.getString('header.title');
	var el = document.getElementsByClassName('lang');
	for (var i = 0; i < el.length; i++) {
		var item = el[i];
		item.innerHTML = window.lang.getString(item.id);
	}
	document.getElementsByClassName('generate-button')[0].innerHTML = window.lang.getString('generator.generate_and_download');
	document.getElementsByClassName('reset-button')[0].innerHTML = window.lang.getString('generator.reset_to_default');
	document.getElementsByClassName('navbar-direction')[0].innerHTML = window.lang.getString('navbar.back');
	document.getElementById('generateButtonLoading').innerHTML = window.lang.getString('generator.generate_button_loading');
}

function setSegmentsPreview(segments) {
	savedSegmentsInfo = segments;
	setSegmentsPreviewVisible(true);
}

function setSegmentsPreviewVisible(visible) {
	if (savedSegmentsInfo == null || savedSegmentsInfo == undefined) {
		visible = false;
	}
	if (visible) {
		document.getElementById('table.' + segmentKeys[0] + '.description').rowSpan = segmentKeys.length;
		document.getElementById('table.' + segmentKeys[0] + '.description').innerHTML = '<span>' + savedSegmentsInfo.replaceAll('\n', '<br>') + '</span>';
		
		for (var i = 1; i < segmentKeys.length; i++) {
			document.getElementById('table.' + segmentKeys[i] + '.description').style.display = 'none';
		}
	} else {
		document.getElementById('table.' + segmentKeys[0] + '.description').rowSpan = 1;
		for (var i = 0; i < segmentKeys.length; i++) {
			var id = 'table.' + segmentKeys[i] + '.description';
			document.getElementById(id).style.display = '';
			document.getElementById(id).innerHTML = window.lang.getString(id);
		}
	}
}

function reset() {
	for (var elementId of formFields) {
        localStorage.removeItem(elementId);
    }
	
	window.location.reload(false);
}

function init() {
	initForm();
	
	const urlParams = new URLSearchParams(window.location.search);
	var lang = urlParams.get('lang');
	if (lang == undefined) {
		lang = 'ru';
	}
	
	window.lang = {
		values: {},
		getString: function(key) {
			var ret = window.lang.values[key];
			if (key == 'header.title') {
				return ret + ' ' + calibrator_version;
			}
			return ret;
		}
	};
	initLang(lang);
	
	setTimeout(function() {
		if (checkGo != undefined && window.lang != undefined) {
			checkGo();
		} else {
			setTimeout(this, 100);
		}
	}, 100);
}
