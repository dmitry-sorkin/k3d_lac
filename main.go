package main

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"syscall/js"
)

const filamentDiameter = 1.75

var (
	// Variables from web interface
	bedX, bedY, zOffset, retractLength, firstLayerLineWidth, lineWidth, layerHeight, initKFactor, endKFactor, segmentHeight, smoothTime                                     float64
	firmware, travelSpeed, hotendTemperature, bedTemperature, retractSpeed, cooling, firstLayerPrintSpeed, fastPrintSpeed, slowPrintSpeed, numSegments, numPerimeters, flow int
	bedProbe, retracted, delta                                                                                                                                              bool
	startGcode, endGcode                                                                                                                                                    string
	// Current variables
	currentCoordinates Point
	currentSpeed       int
	currentE           float64
)

const caliVersion = "v1.4"

type Point struct {
	X float64
	Y float64
	Z float64
}

func main() {
	c := make(chan struct{})
	registerFunctions()
	<-c
}

func registerFunctions() {
	js.Global().Set("generate", js.FuncOf(generate))
	js.Global().Set("checkGo", js.FuncOf(checkJs))
	js.Global().Set("checkSegments", js.FuncOf(checkSegments))
}

func setErrorDescription(doc js.Value, lang js.Value, key string, curErr string, hasErr bool, allowModify bool) {
	if !allowModify {
		return
	}
	el := doc.Call("getElementById", key)
	el.Get("style").Set("display", "")
	el.Set("rowSpan", "1")
	if hasErr {
		el.Set("innerHTML", lang.Call("getString", key).String()+"<br><span class=\"inline-error\">"+curErr+"</span>")
	} else {
		el.Set("innerHTML", lang.Call("getString", key).String())
	}
}

func check(showErrorBox bool, allowModify bool) bool {
	errorString := ""
	doc := js.Global().Get("document")
	lang := js.Global().Get("lang")
	doc.Call("getElementById", "resultContainer").Set("innerHTML", "")

	// Параметры принтера
	curErr := ""
	hasErr := false
	retErr := false

	docBedX, err := parseInputToFloat(doc.Call("getElementById", "k3d_la_bedX").Get("value").String())
	if err != nil {
		curErr, hasErr = lang.Call("getString", "error.bed_size_x.format").String(), true
	} else if docBedX < 100 || docBedX > 1000 {
		curErr, hasErr = lang.Call("getString", "error.bed_size_x.small_or_big").String(), true
	} else {
		bedX = docBedX
	}

	setErrorDescription(doc, lang, "table.bed_size_x.description", curErr, hasErr, allowModify)
	if hasErr {
		errorString = errorString + curErr + "\n"
		hasErr = false
		retErr = true
	}

	docBedY, err := parseInputToFloat(doc.Call("getElementById", "k3d_la_bedY").Get("value").String())
	if err != nil {
		curErr, hasErr = lang.Call("getString", "error.bed_size_y.format").String(), true
	} else if docBedY < 100 || docBedY > 1000 {
		curErr, hasErr = lang.Call("getString", "error.bed_size_y.small_or_big").String(), true
	} else {
		bedY = docBedY
	}

	setErrorDescription(doc, lang, "table.bed_size_y.description", curErr, hasErr, allowModify)
	if hasErr {
		errorString = errorString + curErr + "\n"
		hasErr = false
		retErr = true
	}

	docMarlin := doc.Call("getElementById", "k3d_la_firmwareMarlin").Get("checked").Bool()
	docKlipper := doc.Call("getElementById", "k3d_la_firmwareKlipper").Get("checked").Bool()
	docRRF := doc.Call("getElementById", "k3d_la_firmwareRRF").Get("checked").Bool()
	if docMarlin {
		firmware = 0
	} else if docKlipper {
		firmware = 1
	} else if docRRF {
		firmware = 2
	} else {
		errorString = errorString + lang.Call("getString", "error.firmware.not_set").String() + "\n"
	}

	delta = doc.Call("getElementById", "k3d_la_delta").Get("checked").Bool()

	bedProbe = doc.Call("getElementById", "k3d_la_g29").Get("checked").Bool()

	docTravelSpeed, err := parseInputToInt(doc.Call("getElementById", "k3d_la_travelSpeed").Get("value").String())
	if err != nil {
		curErr, hasErr = lang.Call("getString", "error.travel_speed.format").String(), true
	} else if docTravelSpeed < 10 || docTravelSpeed > 1000 {
		curErr, hasErr = lang.Call("getString", "error.travel_speed.slow_or_fast").String(), true
	} else {
		travelSpeed = docTravelSpeed
	}

	setErrorDescription(doc, lang, "table.travel_speed.description", curErr, hasErr, allowModify)
	if hasErr {
		errorString = errorString + curErr + "\n"
		hasErr = false
		retErr = true
	}

	// Параметры филамента

	docHotTemp, err := parseInputToInt(doc.Call("getElementById", "k3d_la_hotendTemperature").Get("value").String())
	if err != nil {
		curErr, hasErr = lang.Call("getString", "error.hotend_temp.format").String(), true
	} else if docHotTemp < 150 {
		curErr, hasErr = lang.Call("getString", "error.hotend_temp.too_low").String(), true
	} else if docHotTemp > 350 {
		curErr, hasErr = lang.Call("getString", "error.hotend_temp.too_high").String(), true
	} else {
		hotendTemperature = docHotTemp
	}

	setErrorDescription(doc, lang, "table.hotend_temp.description", curErr, hasErr, allowModify)
	if hasErr {
		errorString = errorString + curErr + "\n"
		hasErr = false
		retErr = true
	}

	docBedTemp, err := parseInputToInt(doc.Call("getElementById", "k3d_la_bedTemperature").Get("value").String())
	if err != nil {
		curErr, hasErr = lang.Call("getString", "error.bed_temp.format").String()+err.Error(), true
	} else if docBedTemp > 150 {
		curErr, hasErr = lang.Call("getString", "error.bed_temp.too_high").String(), true
	} else {
		bedTemperature = docBedTemp
	}

	setErrorDescription(doc, lang, "table.bed_temp.description", curErr, hasErr, allowModify)
	if hasErr {
		errorString = errorString + curErr + "\n"
		hasErr = false
		retErr = true
	}

	docCooling, err := parseInputToInt(doc.Call("getElementById", "k3d_la_cooling").Get("value").String())
	if err != nil {
		curErr, hasErr = lang.Call("getString", "error.fan_speed.format").String(), true
	} else {
		docCooling = int(float64(docCooling) * 2.55)
		if docCooling < 0 {
			docCooling = 0
		} else if docCooling > 255 {
			docCooling = 255
		}
		cooling = docCooling
	}
	setErrorDescription(doc, lang, "table.fan_speed.description", curErr, hasErr, allowModify)
	if hasErr {
		errorString = errorString + curErr + "\n"
		hasErr = false
		retErr = true
	}

	retractLength = 1.0

	retractSpeed = 30.0

	docFlow, err := parseInputToInt(doc.Call("getElementById", "k3d_la_flow").Get("value").String())
	if err != nil {
		curErr, hasErr = lang.Call("getString", "error.flow.format").String(), true
	} else if docFlow < 50 || docFlow > 150 {
		curErr, hasErr = lang.Call("getString", "error.flow.low_or_high").String(), true
	} else {
		flow = docFlow
	}
	setErrorDescription(doc, lang, "table.flow.description", curErr, hasErr, allowModify)
	if hasErr {
		errorString = errorString + curErr + "\n"
		hasErr = false
		retErr = true
	}

	// Параметры первого слоя

	docFirstLayerLineWidth, err := parseInputToFloat(doc.Call("getElementById", "k3d_la_firstLayerLineWidth").Get("value").String())
	if err != nil {
		curErr, hasErr = lang.Call("getString", "error.first_line_width.format").String(), true
	} else if docFirstLayerLineWidth < 0.1 || docFirstLayerLineWidth > 2.0 {
		curErr, hasErr = lang.Call("getString", "error.first_line_width.small_or_big").String(), true
	} else {
		firstLayerLineWidth = docFirstLayerLineWidth
	}
	setErrorDescription(doc, lang, "table.first_line_width.description", curErr, hasErr, allowModify)
	if hasErr {
		errorString = errorString + curErr + "\n"
		hasErr = false
		retErr = true
	}

	docFirstLayerPrintSpeed, err := parseInputToInt(doc.Call("getElementById", "k3d_la_firstLayerSpeed").Get("value").String())
	if err != nil {
		curErr, hasErr = lang.Call("getString", "error.first_print_speed.format").String(), true
	} else if docFirstLayerPrintSpeed < 10 || docFirstLayerPrintSpeed > 1000 {
		curErr, hasErr = lang.Call("getString", "error.first_print_speed.slow_or_fast").String(), true
	} else {
		firstLayerPrintSpeed = docFirstLayerPrintSpeed
	}
	setErrorDescription(doc, lang, "table.first_print_speed.description", curErr, hasErr, allowModify)
	if hasErr {
		errorString = errorString + curErr + "\n"
		hasErr = false
		retErr = true
	}

	docZOffset, err := parseInputToFloat(doc.Call("getElementById", "k3d_la_zOffset").Get("value").String())
	if err != nil {
		curErr, hasErr = lang.Call("getString", "error.z_offset.format").String(), true
	} else if docZOffset < -0.5 || docZOffset > 0.5 {
		curErr, hasErr = lang.Call("getString", "error.z_offset.small_or_big").String(), true
	} else {
		zOffset = docZOffset
	}
	setErrorDescription(doc, lang, "table.z_offset.description", curErr, hasErr, allowModify)
	if hasErr {
		errorString = errorString + curErr + "\n"
		hasErr = false
		retErr = true
	}

	// Параметры модели

	docNumPerimeters, err := parseInputToInt(doc.Call("getElementById", "k3d_la_numPerimeters").Get("value").String())
	if err != nil {
		curErr, hasErr = lang.Call("getString", "error.num_perimeters.format").String(), true
	} else if docNumPerimeters < 1 || docNumPerimeters > 5 {
		curErr, hasErr = lang.Call("getString", "error.num_perimeters.small_or_big").String(), true
	} else {
		numPerimeters = docNumPerimeters
	}
	setErrorDescription(doc, lang, "table.num_perimeters.description", curErr, hasErr, allowModify)
	if hasErr {
		errorString = errorString + curErr + "\n"
		hasErr = false
		retErr = true
	}

	docLineWidth, err := parseInputToFloat(doc.Call("getElementById", "k3d_la_lineWidth").Get("value").String())
	if err != nil {
		curErr, hasErr = lang.Call("getString", "error.line_width.format").String(), true
	} else if docLineWidth < 0.1 || docLineWidth > 2.0 {
		curErr, hasErr = lang.Call("getString", "error.line_width.small_or_big").String(), true
	} else {
		lineWidth = docLineWidth
	}
	setErrorDescription(doc, lang, "table.line_width.description", curErr, hasErr, allowModify)
	if hasErr {
		errorString = errorString + curErr + "\n"
		hasErr = false
		retErr = true
	}

	docLayerHeight, err := parseInputToFloat(doc.Call("getElementById", "k3d_la_layerHeight").Get("value").String())
	if err != nil {
		curErr, hasErr = lang.Call("getString", "error.layer_height.format").String(), true
	} else if docLayerHeight < 0.05 || docLayerHeight > 1.2 {
		curErr, hasErr = lang.Call("getString", "error.layer_height.small_or_big").String(), true
	} else {
		layerHeight = docLayerHeight
	}
	setErrorDescription(doc, lang, "table.layer_height.description", curErr, hasErr, allowModify)
	if hasErr {
		errorString = errorString + curErr + "\n"
		hasErr = false
		retErr = true
	}

	docFastPrintSpeed, err := parseInputToInt(doc.Call("getElementById", "k3d_la_fastPrintSpeed").Get("value").String())
	if err != nil {
		curErr, hasErr = lang.Call("getString", "error.fast_segment_speed.format").String(), true
	} else if docFastPrintSpeed < 10 || docFastPrintSpeed > 1000 {
		curErr, hasErr = lang.Call("getString", "error.fast_segment_speed.small_or_big").String(), true
	} else {
		fastPrintSpeed = docFastPrintSpeed
	}
	setErrorDescription(doc, lang, "table.fast_segment_speed.description", curErr, hasErr, allowModify)
	if hasErr {
		errorString = errorString + curErr + "\n"
		hasErr = false
		retErr = true
	}

	docSlowPrintSpeed, err := parseInputToInt(doc.Call("getElementById", "k3d_la_slowPrintSpeed").Get("value").String())
	if err != nil {
		curErr, hasErr = lang.Call("getString", "error.slow_segment_speed.format").String(), true
	} else if docSlowPrintSpeed < 10 || docSlowPrintSpeed > 1000 {
		curErr, hasErr = lang.Call("getString", "error.slow_segment_speed.small_or_big").String(), true
	} else {
		slowPrintSpeed = docSlowPrintSpeed
	}
	setErrorDescription(doc, lang, "table.slow_segment_speed.description", curErr, hasErr, allowModify)
	if hasErr {
		errorString = errorString + curErr + "\n"
		hasErr = false
		retErr = true
	}

	// Параметры калибровки

	docInitKFactor, err := parseInputToFloat(doc.Call("getElementById", "k3d_la_initKFactor").Get("value").String())
	if err != nil {
		curErr, hasErr = lang.Call("getString", "error.init_la.format").String(), true
	} else if docInitKFactor < 0.0 || docInitKFactor > 2.0 {
		curErr, hasErr = lang.Call("getString", "error.init_la.small_or_big").String(), true
	} else {
		initKFactor = docInitKFactor
	}
	setErrorDescription(doc, lang, "table.init_la.description", curErr, hasErr, allowModify)
	if hasErr {
		errorString = errorString + curErr + "\n"
		hasErr = false
		retErr = true
	}

	docEndKFactor, err := parseInputToFloat(doc.Call("getElementById", "k3d_la_endKFactor").Get("value").String())
	if err != nil {
		curErr, hasErr = lang.Call("getString", "error.end_la.format").String(), true
	} else if docEndKFactor < 0.0 || docEndKFactor > 2.0 {
		curErr, hasErr = lang.Call("getString", "error.end_la.small_or_big").String(), true
	} else {
		endKFactor = docEndKFactor
	}
	setErrorDescription(doc, lang, "table.end_la.description", curErr, hasErr, allowModify)
	if hasErr {
		errorString = errorString + curErr + "\n"
		hasErr = false
		retErr = true
	}

	docNumSegment, err := parseInputToInt(doc.Call("getElementById", "k3d_la_numSegments").Get("value").String())
	if err != nil {
		curErr, hasErr = lang.Call("getString", "error.num_segments.format").String(), true
	} else if docNumSegment < 2 || docNumSegment > 100 {
		curErr, hasErr = lang.Call("getString", "error.num_segments.small_or_big").String(), true
	} else {
		numSegments = docNumSegment
	}
	setErrorDescription(doc, lang, "table.num_segments.description", curErr, hasErr, allowModify)
	if hasErr {
		errorString = errorString + curErr + "\n"
		hasErr = false
		retErr = true
	}

	docSegmentHeight, err := parseInputToFloat(doc.Call("getElementById", "k3d_la_segmentHeight").Get("value").String())
	if err != nil {
		curErr, hasErr = lang.Call("getString", "error.segment_height.format").String(), true
	} else if docSegmentHeight < 0.5 || docSegmentHeight > 10.0 {
		curErr, hasErr = lang.Call("getString", "error.segment_height.small_or_big").String(), true
	} else {
		segmentHeight = docSegmentHeight
	}
	setErrorDescription(doc, lang, "table.segment_height.description", curErr, hasErr, allowModify)
	if hasErr {
		errorString = errorString + curErr + "\n"
		hasErr = false
		retErr = true
	}

	docSmoothTime, err := parseInputToFloat(doc.Call("getElementById", "k3d_la_smoothTime").Get("value").String())
	if err != nil {
		curErr, hasErr = lang.Call("getString", "error.smooth_time.format").String(), true
	} else if docSmoothTime < 0.005 || docSmoothTime > 0.2 {
		curErr, hasErr = lang.Call("getString", "error.smooth_time.small_or_big").String(), true
	} else {
		smoothTime = docSmoothTime
	}
	setErrorDescription(doc, lang, "table.smooth_time.description", curErr, hasErr, allowModify)
	if hasErr {
		errorString = errorString + curErr + "\n"
		hasErr = false
		retErr = true
	}

	startGcode = doc.Call("getElementById", "k3d_la_startGcode").Get("value").String()
	endGcode = doc.Call("getElementById", "k3d_la_endGcode").Get("value").String()

	if !showErrorBox {
		return !retErr
	}

	// end check of parameters
	if !retErr {
		println("OK")
		return true
	} else {
		println(errorString)
		js.Global().Call("showError", errorString)
		return false
	}
}

func write(str ...string) {
	for i := 0; i < len(str); i++ {
		js.Global().Call("writeToFile", str[i])
	}
}

func checkSegments(this js.Value, i []js.Value) interface{} {
	if check(false, false) {
		lang := js.Global().Get("lang")
		segmentStr := lang.Call("getString", "generator.segment").String()

		deltaKFactor := math.Abs((endKFactor - initKFactor) / float64(numSegments-1))
		maxKFactor := math.Max(initKFactor, endKFactor)

		// generate calibration parameters
		caliParams := ""
		for i := 0; i < numSegments; i++ {
			caliParams += fmt.Sprintf(segmentStr, numSegments-i, fmt.Sprint(roundFloat(maxKFactor-deltaKFactor*float64(i), 3)))
		}

		js.Global().Call("setSegmentsPreview", caliParams)
	} else {
		js.Global().Call("setSegmentsPreview", js.ValueOf(nil))
		check(false, true)
	}
	return js.ValueOf(nil)
}

func checkJs(this js.Value, i []js.Value) interface{} {
	check(false, true)
	return js.ValueOf(nil)
}

func generate(this js.Value, i []js.Value) interface{} {
	// check and initialize variables
	if !check(true, false) {
		return js.ValueOf(nil)
	}

	// generate calibration parameters
	lang := js.Global().Get("lang")
	segmentStr := lang.Call("getString", "generator.segment").String()

	caliParams := ""
	deltaKFactor := math.Abs((endKFactor - initKFactor) / float64(numSegments-1))
	maxKFactor := math.Max(initKFactor, endKFactor)
	minKFactor := math.Min(initKFactor, endKFactor)
	currentKFactor := minKFactor
	caliParams += "; ====================\n; Поддержите выход новых калибраторов, инструкций и видео!\n;https://donate.stream/dmitrysorkin\n; ====================\n"
	for i := 0; i < numSegments; i++ {
		caliParams += fmt.Sprintf(segmentStr, numSegments-i, fmt.Sprint(roundFloat(maxKFactor-deltaKFactor*float64(i), 3)))
	}

	fileName := fmt.Sprintf("K3D_LA_H%d-B%d_%s-%s_d%s.gcode", hotendTemperature, bedTemperature, fmt.Sprint(roundFloat(initKFactor, 2)), fmt.Sprint(roundFloat(endKFactor, 2)), fmt.Sprint(roundFloat(deltaKFactor, 3)))
	js.Global().Call("beginSaveFile", fileName)

	// gcode initialization
	write("; generated by K3D LA calibration ", js.Global().Get("calibrator_version").String(), "\n",
		"; Written by Dmitry Sorkin @ http://k3d.tech/, Kekht and YTKAB0BP\n",
		fmt.Sprintf(";Bedsize: %s:%s [mm]\n", fmt.Sprint(roundFloat(bedX, 1)), fmt.Sprint(roundFloat(bedY, 1))),
		fmt.Sprintf(";Firmware (0-Marlin, 1-Klipper, 2-RRF): %d\n", firmware),
		fmt.Sprintf(";Z-offset: %s [mm]\n", fmt.Sprint(roundFloat(zOffset, 3))),
		fmt.Sprintf(";Delta: %s\n", strconv.FormatBool(delta)),
		fmt.Sprintf(";G29: %s\n", strconv.FormatBool(bedProbe)),
		fmt.Sprintf(";Temp: %d/%d [°C]\n", hotendTemperature, bedTemperature),
		fmt.Sprintf(";Flow: %d\n", flow),
		fmt.Sprintf(";Fan: %s\n", fmt.Sprint(roundFloat(float64(cooling)/2.55, 1))),
		fmt.Sprintf(";Line width: %s [mm]\n", fmt.Sprint(roundFloat(lineWidth, 2))),
		fmt.Sprintf(";First layer line width: %s [mm]\n", fmt.Sprint(roundFloat(lineWidth, 2))),
		fmt.Sprintf(";Layer height: %s [mm]\n", fmt.Sprint(roundFloat(layerHeight, 2))),
		fmt.Sprintf(";Fast print speed: %d [mm/s]\n", fastPrintSpeed),
		fmt.Sprintf(";Slow print speed: %d [mm/s]\n", slowPrintSpeed),
		fmt.Sprintf(";First layer print speed: %d [mm/s]\n", firstLayerPrintSpeed),
		fmt.Sprintf(";Travel speed: %d [mm/s]\n", travelSpeed),
		fmt.Sprintf(";Segment height: %s [mm]\n", fmt.Sprint(roundFloat(segmentHeight, 2))),
		caliParams)

	var g29str string
	if bedProbe {
		g29str = "G29"
	} else {
		g29str = ""
	}
	replacer := strings.NewReplacer("$BEDTEMP", strconv.Itoa(bedTemperature), "$HOTTEMP", strconv.Itoa(hotendTemperature), "$G29", g29str, "$FLOW", strconv.Itoa(flow))
	write(replacer.Replace(startGcode), "\n")

	write("M82\n", "M106 S0\n")

	// generate first layer
	var bedCenter Point
	if delta {
		bedCenter.X, bedCenter.Y, bedCenter.Z = 0, 0, layerHeight
	} else {
		bedCenter.X, bedCenter.Y, bedCenter.Z = bedX/2, bedY/2, layerHeight
	}
	currentE = 0
	currentSpeed = firstLayerPrintSpeed
	currentCoordinates.X, currentCoordinates.Y, currentCoordinates.Z = 0, 0, 0

	// move to layer height to avoid nozzle striking at bed
	write(fmt.Sprintf("G1 Z%s\n", fmt.Sprint(roundFloat(layerHeight+zOffset, 2))))

	// make printer think, that he is on layerHeight
	write(fmt.Sprintf("G92 Z%s\n", fmt.Sprint(roundFloat(layerHeight, 2))))
	currentCoordinates.Z = layerHeight

	// purge nozzle
	modelWidth := 40.0
	var purgeStart Point
	purgeStart.X, purgeStart.Y, purgeStart.Z = bedCenter.X-bedX/2+15.0, bedCenter.Y-modelWidth-10.0, currentCoordinates.Z
	purgeTwo := purgeStart
	purgeTwo.X = bedCenter.X + bedX/2 - 15.0
	purgeThree := purgeTwo
	purgeThree.Y += firstLayerLineWidth
	purgeEnd := purgeThree
	purgeEnd.X = purgeStart.X

	// move to start of purge
	write(generateMove(currentCoordinates, purgeStart, 0.0, travelSpeed)...)

	// add purge to gcode
	write(generateMove(currentCoordinates, purgeTwo, firstLayerLineWidth, firstLayerPrintSpeed)...)
	write(generateMove(currentCoordinates, purgeThree, firstLayerLineWidth, firstLayerPrintSpeed)...)
	write(generateMove(currentCoordinates, purgeEnd, firstLayerLineWidth, firstLayerPrintSpeed)...)

	// generate raft trajectory
	trajectory := generateZigZagTrajectory(bedCenter, firstLayerLineWidth, modelWidth+10.0)

	// move to start of raft
	write(generateRetraction())
	write(generateMove(currentCoordinates, trajectory[0], 0.0, travelSpeed)...)
	write(generateDeretraction())

	// print raft
	for i := 1; i < len(trajectory); i++ {
		write(generateMove(currentCoordinates, trajectory[i], firstLayerLineWidth, firstLayerPrintSpeed)...)
	}

	// set LA for first segment
	write(generateLACommand(currentKFactor))

	// generate model
	layersPerSegment := int(segmentHeight / layerHeight)
	for i := 1; i < numSegments*layersPerSegment; i++ {
		// add layer start comment
		write(fmt.Sprintf(";layer #%s\n", fmt.Sprint(roundFloat(currentCoordinates.Z/layerHeight, 0))))

		// change fan speed
		if i < 4 {
			write(fmt.Sprintf("M106 S%s\n", fmt.Sprint(roundFloat(float64(cooling*i/3), 0))))
		}

		// modify print settings if switching segments
		addition := 0.0
		if i%layersPerSegment == 0 {
			currentKFactor += deltaKFactor
			write(generateLACommand(currentKFactor))
			addition = lineWidth / 2
		} else {
			addition = 0
		}

		// move to start of new layer
		layerStart := bedCenter
		if i%layersPerSegment == 0 {
			layerStart.Y += (modelWidth - lineWidth/2) / 2
		} else {
			layerStart.Y += (modelWidth - lineWidth) / 2
		}
		layerStart.Z = currentCoordinates.Z + layerHeight
		write(generateMove(currentCoordinates, layerStart, 0.0, travelSpeed)...)
		currentCoordinates.Z = layerStart.Z
		// generate layer gcode
		for j := 0; j < numPerimeters; j++ {
			// calc lines parameters
			currentModelWidth := modelWidth + addition - lineWidth*2*float64(j+1)
			rightShortLine := 20.0
			rightLongLine := (currentModelWidth - rightShortLine) / 2
			frontShortLine := 2.0
			frontLongLine := (currentModelWidth - frontShortLine) / 2
			leftShortLine := 0.2
			leftLongLine := (currentModelWidth - leftShortLine) / 2
			// print back line's right part
			write(generateRelativeMove(currentModelWidth/2, 0, 0, lineWidth, fastPrintSpeed)...)
			// print right line
			write(generateRelativeMove(0, -rightLongLine, 0, lineWidth, fastPrintSpeed)...)
			write(generateRelativeMove(0, -rightShortLine, 0, lineWidth, slowPrintSpeed)...)
			write(generateRelativeMove(0, -rightLongLine, 0, lineWidth, fastPrintSpeed)...)
			// print front line
			write(generateRelativeMove(-frontLongLine, 0, 0, lineWidth, fastPrintSpeed)...)
			write(generateRelativeMove(-frontShortLine, 0, 0, lineWidth, slowPrintSpeed)...)
			write(generateRelativeMove(-frontLongLine, 0, 0, lineWidth, fastPrintSpeed)...)
			// print left line
			write(generateRelativeMove(0, leftLongLine, 0, lineWidth, fastPrintSpeed)...)
			write(generateRelativeMove(0, leftShortLine, 0, lineWidth, slowPrintSpeed)...)
			write(generateRelativeMove(0, leftLongLine, 0, lineWidth, fastPrintSpeed)...)
			// print back line left part
			write(generateRelativeMove(currentModelWidth/2, 0, 0, lineWidth, fastPrintSpeed)...)
			// move to start of next perimeter if it exists
			if j != numPerimeters-1 {
				write(generateRelativeMove(0, -lineWidth, 0, 0.0, fastPrintSpeed)...)
			}
		}
	}

	// end gcode
	write(endGcode)

	// write calibration parameters to resultContainer
	js.Global().Call("showError", caliParams)

	// save file
	js.Global().Call("finishFile")

	return js.ValueOf(nil)
}

func generateLACommand(kFactor float64) string {
	if firmware == 0 {
		return fmt.Sprintf("M900 K%s\n", fmt.Sprint(roundFloat(kFactor, 3)))
	} else if firmware == 1 {
		return fmt.Sprintf("SET_PRESSURE_ADVANCE ADVANCE=%s SMOOTH_TIME=%s\n", fmt.Sprint(roundFloat(kFactor, 3)), fmt.Sprint(roundFloat(smoothTime, 3)))
	} else if firmware == 2 {
		return fmt.Sprintf("M572 D0 S%s\n", fmt.Sprint(roundFloat(kFactor, 3)))
	}

	return ";no firmware information\n"
}

func generateRelativeMove(x, y, z, width float64, speed int) []string {
	endPoint := currentCoordinates
	endPoint.X += x
	endPoint.Y += y
	endPoint.Z += z
	return generateMove(currentCoordinates, endPoint, width, speed)
}

func generateMove(start, end Point, width float64, speed int) []string {
	// create move
	move := make([]string, 0, 1)

	// create G1 command
	command := "G1"

	// add X
	if end.X != start.X {
		command += fmt.Sprintf(" X%s", fmt.Sprint(roundFloat(end.X, 2)))
	}

	// add Y
	if end.Y != start.Y {
		command += fmt.Sprintf(" Y%s", fmt.Sprint(roundFloat(end.Y, 2)))
	}

	// add Z
	if end.Z != start.Z {
		command += fmt.Sprintf(" Z%s", fmt.Sprint(roundFloat(end.Z, 2)))
	}

	// add E
	if width > 0 && math.Sqrt(float64(math.Pow((end.X-start.X), 2)+math.Pow((end.Y-start.Y), 2))) > 0.8 {
		newE := currentE + calcExtrusion(start, end, width)
		command += fmt.Sprintf(" E%s", fmt.Sprint(roundFloat(newE, 4)))
		currentE = newE
	}

	// add F
	command += fmt.Sprintf(" F%d", speed*60)
	currentSpeed = speed

	// add G1 to move
	move = append(move, command+"\n")
	currentCoordinates = end

	return move
}

func calcExtrusion(start, end Point, width float64) float64 {
	lineLength := math.Sqrt(float64(math.Pow((end.X-start.X), 2) + math.Pow((end.Y-start.Y), 2)))
	extrusion := width * layerHeight * lineLength * 4 / math.Pi / math.Pow(filamentDiameter, 2)
	return extrusion
}

func generateZigZagTrajectory(towerCenter Point, lineWidth, raftWidth float64) []Point {
	sideLength := raftWidth - lineWidth
	pointsOnOneSide := int(sideLength / (lineWidth * math.Sqrt(2)))
	pointsOnOneSide = pointsOnOneSide - (pointsOnOneSide-1)%2
	pointSpacing := sideLength / float64(pointsOnOneSide-1)
	firstLayerLineWidth = pointSpacing / math.Sqrt(2)

	totalPoints := pointsOnOneSide*4 - 4
	unsortedPoints := make([]Point, totalPoints)

	minX := towerCenter.X - sideLength/2
	minY := towerCenter.Y - sideLength/2
	maxX := towerCenter.X + sideLength/2
	maxY := towerCenter.Y + sideLength/2

	// Generate unsorted slice of points clockwise
	for i := 0; i <= pointsOnOneSide-1; i++ {
		unsortedPoints[i].X = minX + pointSpacing*float64(i)
		unsortedPoints[i].Y = maxY
	}
	for i := 1; i <= pointsOnOneSide-1; i++ {
		unsortedPoints[pointsOnOneSide+i-1].X = maxX
		unsortedPoints[pointsOnOneSide+i-1].Y = maxY - pointSpacing*float64(i)
	}
	for i := 1; i <= pointsOnOneSide-1; i++ {
		unsortedPoints[pointsOnOneSide*2+i-2].X = maxX - pointSpacing*float64(i)
		unsortedPoints[pointsOnOneSide*2+i-2].Y = minY
	}
	for i := 1; i < pointsOnOneSide-1; i++ {
		unsortedPoints[pointsOnOneSide*3+i-3].X = minX
		unsortedPoints[pointsOnOneSide*3+i-3].Y = minY + pointSpacing*float64(i)
	}

	// add Z coordinates

	for i := 1; i < len(unsortedPoints); i++ {
		unsortedPoints[i].Z = currentCoordinates.Z
	}

	// Sort points to make zigzag moves
	trajectory := make([]Point, len(unsortedPoints))

	trajectory[0] = unsortedPoints[0]
	trajectory[1] = unsortedPoints[len(unsortedPoints)-1]
	trajectory[2] = unsortedPoints[1]
	trajectory[3] = unsortedPoints[2]
	for i := 4; i < len(unsortedPoints); i = i + 4 {
		j := int(i / 2)
		trajectory[i] = unsortedPoints[len(unsortedPoints)-j]
		trajectory[i+1] = unsortedPoints[len(unsortedPoints)-j-1]
		trajectory[i+2] = unsortedPoints[j+1]
		trajectory[i+3] = unsortedPoints[j+2]
	}

	for i := 0; i < len(trajectory); i++ {
		trajectory[i].Z = currentCoordinates.Z
	}

	return trajectory
}

func generateRetraction() string {
	if retracted {
		fmt.Println("Called retraction, but already retracted")
		return ""
	} else {
		retracted = true
		currentSpeed = retractSpeed
		return fmt.Sprintf("G1 E%s F%d\n", fmt.Sprint(roundFloat(currentE-retractLength, 2)), retractSpeed*60)
	}
}

func generateDeretraction() string {
	if retracted {
		retracted = false
		currentSpeed = retractSpeed
		return fmt.Sprintf("G1 E%s F%d\n", fmt.Sprint(roundFloat(currentE, 2)), retractSpeed*60)
	} else {
		fmt.Println("Called deretraction, but not retracted")
		return ""
	}
}

func roundFloat(val float64, precision uint) float64 {
	ratio := math.Pow(10, float64(precision))
	return math.Round(val*ratio) / ratio
}

func parseInputToFloat(val string) (float64, error) {
	f, err := strconv.ParseFloat(strings.ReplaceAll(val, ",", "."), 64)
	if err != nil {
		println(err.Error())
	}
	return f, err
}

func parseInputToInt(val string) (int, error) {
	f, err := parseInputToFloat(val)
	return int(roundFloat(f, 0)), err
}
