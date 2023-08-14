const calibrator_version = 'v1.4';
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
    "k3d_la_numSegments",
	"k3d_la_startGcode",
	"k3d_la_endGcode"
];
var segmentFields = [
    "k3d_la_initKFactor",
	"k3d_la_endKFactor",
	"k3d_la_numSegments"
];
var segmentKeys = [
	"init_la",
	"end_la",
	"num_segments"
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
			values['header.title'] = 'K3D Linear Advance calibrator';
			values['header.language'] = 'Language: ';
			values['header.useful_links'] = 'Useful links: ';
			values['header.instruction'] = 'Instructions for use';
			values['header.width_not_changing'] = 'What to do if the thickness of the central section does not change?';
			
			values['table.header.parameter'] = 'Parameter';
			values['table.header.value'] = 'Value';
			values['table.header.description'] = 'Description';
			
			values['table.bed_size_x.title'] = 'Bed size X';
			values['table.bed_size_x.description'] = '[mm] For cartesian printers - maximum X coordinate<br>For delta-printers - <b>bed diameter</b>';
			values['table.bed_size_y.title'] = 'Bed size Y';
			values['table.bed_size_y.description'] = '[mm] For cartesian printers - maximum Y coordinate<br>For delta-printers - <b>bed diameter</b>';
			values['table.firmware.title'] = 'Firmware';
			values['table.firmware.description'] = 'Firmware installed on your printer. If you don\'t know, then it\'s probably Marlin';
			values['table.delta.title'] = 'Origin at the center of the bed';
			values['table.delta.description'] = 'Must be disabled for cartesian printers, enabled for deltas';
			values['table.bed_probe.title'] = 'Bed auto-calibration';
			values['table.bed_probe.description'] = 'Enables bed auto-calibration before printing (G29)? If you don\'t have bed probe, then leave it off.';
			values['table.travel_speed.title'] = 'Travel speed';
			values['table.travel_speed.description'] = '[mm/s] The speed at which movements will occur without extrusion';
			values['table.hotend_temp.title'] = 'Hotend temperature';
			values['table.hotend_temp.description'] = '[°C] The temperature to which to heat the hotend before printing';
			values['table.bed_temp.title'] = 'Bed temperature';
			values['table.bed_temp.description'] = '[°C] The temperature to which the bed must be heated before printing. The bed will heat up until parking and auto-calibration.';
			values['table.fan_speed.title'] = 'Fan speed';
			values['table.fan_speed.description'] = '[%] Fan speed in percent. In order for the temperature of the hot end not to drop sharply when the fan is turned on, the airflow will be turned off on the 1st layer. On layers 2-4, the fan speed will increase in steps to the specified value';
			values['table.flow.title'] = 'Flow';
			values['table.flow.description'] = '[%] Flow in percents. Needed to compensate for over- or under-extrusion';
			values['table.first_line_width.title'] = 'First layer line width';
			values['table.first_line_width.description'] = '[mm] The line width at which the raft will be printed under the towers. In general, it is recommended to set 150% of the nozzle diameter';
			values['table.first_print_speed.title'] = 'First layer print speed';
			values['table.first_print_speed.description'] = '[mm/s] The speed at which the raft under the towers will be printed';
			values['table.z_offset.title'] = 'Z-offset';
			values['table.z_offset.description'] = '[mm] Offset the entire model vertically. It is necessary to compensate for too thin / thick first layer calibration. Leave zero in general.';
			values['table.num_perimeters.title'] = 'Number of perimeters';
			values['table.num_perimeters.description'] = 'The number of perimeters for the main body of the calibration model. For near-zero shrinkage filaments (PLA, some composites) 1-2. For high shrinkage filaments (ABS and similar) 2+. For flexes 2-4 depending on their rigidity and the desired height of the tower';
			values['table.line_width.title'] = 'Line width';
			values['table.line_width.description'] = '[mm] The line width at which the towers will be printed. In general, it is recommended to set equal to the nozzle diameter';
			values['table.layer_height.title'] = 'Layer height';
			values['table.layer_height.description'] = '[mm] The thickness of the layers of the entire model. In general, 50% of the line width';
			values['table.fast_segment_speed.title'] = 'Speed of fast sections';
			values['table.fast_segment_speed.description'] = '[mm/s] The speed at which fast sections will be printed. It is better to specify high values (100-150)';
			values['table.slow_segment_speed.title'] = 'Speed of slow sections';
			values['table.slow_segment_speed.description'] = '[mm/s] The speed at which slow sections will be printed. It is better to specify low values (10-30)';
			values['table.init_la.title'] = 'Initial value of the LA coefficient';
			values['table.init_la.description'] = 'What is the value of the k-factor to start the calibration. Rounded up to 3 decimal places';
			values['table.end_la.title'] = 'Final value of the LA coefficient';
			values['table.end_la.description'] = 'To what value of the k-factor to calibrate. Rounded to 3 decimal places after the separator. For direct extruders, 0.2 is usually enough, for bowdens 1.5';
			values['table.num_segments.title'] = 'Number of segments';
			values['table.num_segments.description'] = 'The number of tower segments. During the segment, the LA coefficient remains unchanged. Segments are visually separated to simplify model analysis';
			values['table.segment_height.title'] = 'Segment height';
			values['table.segment_height.description'] = '[mm] The height of one segment of the tower. For example, if the height of the segment is 3mm, and the number of segments is 10, then the height of the entire tower will be 30mm';
			values['table.start_gcode.title'] = 'Start G-Code';
			values['table.start_gcode.description'] = 'The code that is executed before test. Change at your own risk! List of possible placeholders:<br><b>$BEDTEMP</b> - bed temperature<br><b>$HOTTEMP</b> - hotend temperature<br><b>$G29</b> - bed heightmap command<br><b>$FLOW</b> - flow';
			values['table.end_gcode.title'] = 'End G-Code';
			values['table.end_gcode.description'] = 'The code that is executed after the test. Change at your own risk!';
			
			values['generator.generate_and_download'] = 'Generate and download';		
			values['generator.generate_button_loading'] = 'Generator loading...';
			values['generator.segment'] = '; Segment %d: K-Factor: %s\n';
			values['generator.reset_to_default'] = 'Reset settings';
			
			values['navbar.back'] = ' Back ';
			values['navbar.site'] = 'Site';
			
			values['error.bed_size_x.format'] = 'Bed size Х - format error';
			values['error.bed_size_x.small_or_big'] = 'Bed size X is incorrect (less than 100 or greater than 1000 mm)';
			values['error.bed_size_y.format'] = 'Bed size Y - format error';
			values['error.bed_size_y.small_or_big'] = 'Bed size Y is incorrect (less than 100 or greater than 1000 mm)';
			values['error.hotend_temp.format'] = 'Hotend temperature - format error';
			values['error.hotend_temp.too_low'] = 'Hotend temperature is too low';
			values['error.hotend_temp.too_high'] = 'Hotend temperature is too high';
			values['error.bed_temp.format'] = 'Bed temperature - format error: ';
			values['error.bed_temp.too_high'] = 'Bed temperature is too high';
			values['error.fan_speed.format'] = 'Fan speed - format error';
			values['error.line_width.format'] = 'Line width - format error';
			values['error.line_width.small_or_big'] = 'Wrong line width (less than 0.1 or greater than 2.0 mm)';
			values['error.first_line_width.format'] = 'First layer line width - format error';
			values['error.first_line_width.small_or_big'] = 'Wrong first line width (less than 0.1 or greater than 2.0 mm)';
			values['error.layer_height.format'] = 'Layer height - format error';
			values['error.layer_height.small_or_big'] = 'Wrong layer height (less than 0.05 mm or greater than 75% from line width)';
			values['error.first_print_speed.format'] = 'First layer print speed - format error';
			values['error.first_print_speed.slow_or_fast'] = 'Wrong first layer print speed (less than 10 or greater than 1000 mm/s)';
			values['error.travel_speed.format'] = 'Travel speed - format error';
			values['error.travel_speed.slow_or_fast'] = 'Wrong travel speed (less than 10 or greater than 1000 mm/s)';
			values['error.num_segments.format'] = 'Number of segments - format error';
			values['error.num_segments.slow_or_fast'] = 'Wrong number of segments (less than 2 or greater than 100)';
			values['error.segment_height.format'] = 'Segment height - format error';
			values['error.segment_height.small_or_big'] = 'Wrong segment height (less than 0.5 or greater than 10 mm)';
			values['error.z_offset.format'] = 'Z-offset - format error';
			values['error.z_offset.small_or_big'] = 'Offset value is wrong (less than -0.5 or more than 0.5 mm)';
			values['error.flow.format'] = 'Flow - format error';
			values['error.flow.low_or_high'] = 'Value error: flow should be from 50 to 150%';
			values['error.firmware.not_set'] = 'Format error: firmware not set';
			values['error.num_perimeters.format'] = 'Number of perimeters - format error';
			values['error.num_perimeters.small_or_big'] = 'Value error: number of perimeters must be between 1 and 5';
			values['error.fast_segment_speed.format'] = 'Speed of fast sections - format Error';
			values['error.fast_segment_speed.small_or_big'] = 'The print speed of fast sections is incorrect (less than 10 or more than 1000 mm/s)';
			values['error.slow_segment_speed.format'] = 'Speed of slow sections - format error';
			values['error.slow_segment_speed.small_or_big'] = 'The print speed of slow sections is incorrect (less than 10 or more than 1000 mm/s)';
			values['error.init_la.format'] = 'Initial LA coefficient - format error';
			values['error.init_la.small_or_big'] = 'The initial value of the LA coefficient is incorrect (less than 0.0 or greater than 2.0)';
			values['error.end_la.format'] = 'Final LA coefficient - format error';
			values['error.end_la.small_or_big'] = 'The final value of the LA coefficient is incorrect (less than 0.0 or greater than 2.0)';
			break;
		case 'ru':
			values['header.title'] = 'K3D калибровщик Linear Advance';
			values['header.language'] = 'Язык: ';
			values['header.useful_links'] = 'Полезные ссылки:';
			values['header.instruction'] = 'Инструкция по использованию';
			values['header.width_not_changing'] = 'Что делать, если толщина центрального участка не меняется?';
			
			values['table.header.parameter'] = 'Параметр';
			values['table.header.value'] = 'Значение';
			values['table.header.description'] = 'Описание';
			
			values['table.bed_size_x.title'] = 'Размер стола по X';
			values['table.bed_size_x.description'] = '[мм] Для декартовых принтеров - максимальная координата по оси X<br>Для дельта-принтеров - <b>диаметр стола</b>';
			values['table.bed_size_y.title'] = 'Размер стола по Y';
			values['table.bed_size_y.description'] = '[мм] Для декартовых принтеров - максимальная координата по оси Y<br>Для дельта-принтеров - <b>диаметр стола</b>';
			values['table.firmware.title'] = 'Прошивка';
			values['table.firmware.description'] = 'Прошивка, установленная на вашем принтере. Если не знаете, то, скорее всего, Marlin';
			values['table.delta.title'] = 'Начало координат в центре стола';
			values['table.delta.description'] = 'Для декартовых принтеров должно быть выключено, для дельт включено';
			values['table.bed_probe.title'] = 'Автокалибровка стола';
			values['table.bed_probe.description'] = 'Надо ли делать автокалибровку стола перед печатью (G29)? Если у вас нет датчика автокалибровки, то оставляйте выключенным';
			values['table.travel_speed.title'] = 'Скорость перемещений';
			values['table.travel_speed.description'] = '[мм/с] Скорость, с которой будут происходить перемещения без экструдирования';
			values['table.hotend_temp.title'] = 'Температура хотэнда';
			values['table.hotend_temp.description'] = '[°C] До прогрева стола хотэнд будет нагрет до 150 градусов. После полного нагрева стола хотэнд догреется до указанной температуры';
			values['table.bed_temp.title'] = 'Температура стола';
			values['table.bed_temp.description'] = '[°C] Температура, до которой нагреть стол перед печатью. Стол будет нагрет до выполнения парковки и автокалибровки стола';
			values['table.fan_speed.title'] = 'Скорость вентилятора';
			values['table.fan_speed.description'] = '[%] Обороты вентилятора в процентах. Для того, чтобы температура хотэнда резко не упала при включении вентилятора, на 1 слое обдув будет выключен. На 2-4 слоях скорость вращения вентиляторов будет ступенчато увелиичиваться до указанного значения';
			values['table.flow.title'] = 'Поток';
			values['table.flow.description'] = '[%] Поток в процентах. Нужен для компенсации пере- или недоэкструзии';
			values['table.first_line_width.title'] = 'Ширина линии первого слоя';
			values['table.first_line_width.description'] = '[мм] Ширина линий, с которой будет напечатана подложка под моделью. В общем случае рекомендуется выставить 150% от диаметра сопла';
			values['table.first_print_speed.title'] = 'Скорость печати первого слоя';
			values['table.first_print_speed.description'] = '[мм/с] Скорость, с которой будет напечатана подложка';
			values['table.z_offset.title'] = 'Z-offset';
			values['table.z_offset.description'] = '[мм] Смещение всей модели по вертикали. Нужно чтобы компенсировать слишком тонкую/толстую калибровку первого слоя. В общем случае оставьте ноль';
			values['table.num_perimeters.title'] = 'Количество периметров';
			values['table.num_perimeters.description'] = 'Количество периметров для основного тела калибровочной модели. Для филаментов с околонулевой усадкой (PLA, некоторые композиты) 1-2. Для филаментов с сильной усадкой (ABS и подобные) 2+. Для флексов 2-4 в зависимости от их жесткости и желаемой высоты башенки';
			values['table.line_width.title'] = 'Ширина линии';
			values['table.line_width.description'] = '[мм] Ширина линий, с которой будут напечатаны башенки. В общем случае рекомендуется выставить равной диаметру сопла';
			values['table.layer_height.title'] = 'Толщина слоя';
			values['table.layer_height.description'] = '[мм] Толщина слоёв всей модели. В общем случае 50% от ширины линии';
			values['table.fast_segment_speed.title'] = 'Скорость быстрых участков';
			values['table.fast_segment_speed.description'] = '[мм/с] Скорость, с которой будут печататься быстрые участки. Лучше указать высокие значения (100-150)';
			values['table.slow_segment_speed.title'] = 'Скорость медленных участков';
			values['table.slow_segment_speed.description'] = '[мм/с] Скорость, с которой будут печататься медленные участки. Лучше указать низкие значения (10-30)';
			values['table.init_la.title'] = 'Начальное значение коэффициента LA';
			values['table.init_la.description'] = 'С какого значения к-фактора начать калибровку. Округляется до 3 знака после разделителя';
			values['table.end_la.title'] = 'Конечное значение коэффициента LA';
			values['table.end_la.description'] = 'До какого значения к-фактора проводить калибровку. Округляется до 3 знака после разделителя. Для директ экструдеров обычно хватает 0.2, для боуденов 1.5';
			values['table.num_segments.title'] = 'Количество сегментов';
			values['table.num_segments.description'] = 'Количество сегментов башенки. В течение сегмента коэффициент LA остаётся неизменным. Сегменты визуально разделены для упрощения анализа модели';
			values['table.segment_height.title'] = 'Высота сегмента';
			values['table.segment_height.description'] = '[мм] Высота одного сегмента башенки. К примеру, если высота сегмента 3мм, а количество сегментов 10, то высота всей башенки будет 30мм';
			values['table.start_gcode.title'] = 'Начальный G-код';
			values['table.start_gcode.description'] = 'Код, выполняемый перед печатью теста. Менять на свой страх и риск! Список возможных плейсхолдеров:<br><b>$BEDTEMP</b> - температура стола<br><b>$HOTTEMP</b> - температура хотэнда<br><b>$G29</b> - команда на снятие карты высот стола<br><b>$FLOW</b> - поток';
			values['table.end_gcode.title'] = 'Конечный G-код';
			values['table.end_gcode.description'] = 'Код, выполняемый после печати теста. Менять на свой страх и риск!';
			
			values['generator.generate_and_download'] = 'Генерировать и скачать';		
			values['generator.generate_button_loading'] = 'Генератор загружается...';		
			values['generator.segment'] = '; Сегмент %d: K-Factor: %s\n';
			values['generator.reset_to_default'] = 'Сбросить настройки';
			
			values['navbar.back'] = ' Назад ';
			values['navbar.site'] = 'Сайт';
			
			values['error.bed_size_x.format'] = 'Размер оси Х - ошибка формата';
			values['error.bed_size_x.small_or_big'] = 'Размер стола по X указан неверно (меньше 100 или больше 1000 мм)';
			values['error.bed_size_y.format'] = 'Размер оси Y - ошибка формата';
			values['error.bed_size_y.small_or_big'] = 'Размер стола по Y указан неверно (меньше 100 или больше 1000 мм)';
			values['error.hotend_temp.format'] = 'Температура хотэнда - ошибка формата';
			values['error.hotend_temp.too_low'] = 'Температура хотэнда слишком низкая';
			values['error.hotend_temp.too_high'] = 'Температура хотэнда слишком высокая';
			values['error.bed_temp.format'] = 'Температура стола - ошибка формата: ';
			values['error.bed_temp.too_high'] = 'Температура стола слишком высокая';
			values['error.fan_speed.format'] = 'Скорость вентилятора - ошибка формата';
			values['error.line_width.format'] = 'Ширина линии - ошибка формата';
			values['error.line_width.small_or_big'] = 'Неправильная ширина линии (меньше 0.1 или больше 2.0 мм)';
			values['error.first_line_width.format'] = 'Ширина линии первого слоя - ошибка формата';
			values['error.first_line_width.small_or_big'] = 'Неправильная ширина линии первого слоя (меньше 0.1 или больше 2.0 мм)';
			values['error.layer_height.format'] = 'Высота слоя - ошибка формата';
			values['error.layer_height.small_or_big'] = 'Толщина слоя неправильная (меньше 0.05 или больше 1.2 мм)';
			values['error.first_print_speed.format'] = 'Скорость печати первого слоя - ошибка формата';
			values['error.first_print_speed.slow_or_fast'] = 'Скорость печати первого слоя неправильная (меньше 10 или больше 1000 мм/с)';
			values['error.travel_speed.format'] = 'Скорость перемещений - ошибка формата';
			values['error.travel_speed.slow_or_fast'] = 'Скорость перемещений неправильная (меньше 10 или больше 1000 мм/с)';
			values['error.num_segments.format'] = 'Количество сегментов - ошибка формата';
			values['error.num_segments.small_or_big'] = 'Количество сегментов неправильное (меньше 2 или больше 100)';
			values['error.segment_height.format'] = 'Высота сегмента - ошибка формата';
			values['error.segment_height.small_or_big'] = 'Высота сегмента неправильная (меньше 0.5 или больше 10 мм)';
			values['error.z_offset.format'] = 'Z-offset - ошибка формата';
			values['error.z_offset.small_or_big'] = 'Значение оффсета неправильно (меньше -0.5 или больше 0.5 мм)';
			values['error.flow.format'] = 'Поток - ошибка формата';
			values['error.flow.low_or_high'] = 'Ошибка значения: поток должен быть от 50 до 150%';
			values['error.firmware.not_set'] = 'Ошибка формата: не выбрана прошивка';
			values['error.num_perimeters.format'] = 'Количество периметров - ошибка формата';
			values['error.num_perimeters.small_or_big'] = 'Ошибка значения: количество периметров должно быть от 1 до 5';
			values['error.fast_segment_speed.format'] = 'Скорость печати быстрых участков - ошибка формата';
			values['error.fast_segment_speed.small_or_big'] = 'Скорость печати быстрых участков неверная (меньше 10 или больше 1000 мм/с)';
			values['error.slow_segment_speed.format'] = 'Скорость печати медленных участков - ошибка формата';
			values['error.slow_segment_speed.small_or_big'] = 'Скорость печати медленных участков неверная (меньше 10 или больше 1000 мм/с)';
			values['error.init_la.format'] = 'Начальное значение коэффициента LA - ошибка формата';
			values['error.init_la.small_or_big'] = 'Начальное значение коэффициента LA неверное (меньше 0.0 или больше 2.0)';
			values['error.end_la.format'] = 'Конечное значение коэффициента LA - ошибка формата';
			values['error.end_la.small_or_big'] = 'Конечное значение коэффициента LA неверное (меньше 0.0 или больше 2.0)';
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