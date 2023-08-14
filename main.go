package main

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"syscall/js"
)

const filamentDiameter = 1.75
const retractLength = 2.0
const retractSpeed = 20
const minLineLength = 20.0
const angle = 90.0
const spacing = 1.0

var (
	// Variables from web interface
	bedX, bedY, zOffset, firstLayerLineWidth, lineWidth, layerHeight, la, modelHeight                                                                                       float64
	firmware, travelSpeed, hotendTemperature, bedTemperature, cooling, firstLayerSpeed, printSpeed, numSegments, flow, slowAcceleration, startAcceleration, endAcceleration int
	bedProbe, retracted, delta                                                                                                                                              bool
	startGcode, endGcode                                                                                                                                                    string
	// Current variables
	currentCoordinates Point
	currentSpeed       int
	currentE           float64
)

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
		curErr, hasErr = lang.Call("getString", "error.bed_size_x.value").String(), true
	} else {
		bedX = docBedX
	}

	setErrorDescription(doc, lang, "table.bed_size_x.description", curErr, hasErr, allowModify)
	if hasErr {
		errorString = errorString + curErr + "\n"
		hasErr = false
		retErr = true
	}

	docBedY, err := parseInputToFloat(doc.Call("getElementById", "k3d_smc_bedY").Get("value").String())
	if err != nil {
		curErr, hasErr = lang.Call("getString", "error.bed_size_y.format").String(), true
	} else if docBedY < 100 || docBedY > 1000 {
		curErr, hasErr = lang.Call("getString", "error.bed_size_y.value").String(), true
	} else {
		bedY = docBedY
	}

	setErrorDescription(doc, lang, "table.bed_size_y.description", curErr, hasErr, allowModify)
	if hasErr {
		errorString = errorString + curErr + "\n"
		hasErr = false
		retErr = true
	}

	docMarlin := doc.Call("getElementById", "k3d_smc_firmwareMarlin").Get("checked").Bool()
	docKlipper := doc.Call("getElementById", "k3d_smc_firmwareKlipper").Get("checked").Bool()
	docRRF := doc.Call("getElementById", "k3d_smc_firmwareRRF").Get("checked").Bool()
	if docMarlin {
		firmware = 0
	} else if docKlipper {
		firmware = 1
	} else if docRRF {
		firmware = 2
	} else {
		errorString = errorString + lang.Call("getString", "error.firmware.not_set").String() + "\n"
	}

	delta = doc.Call("getElementById", "k3d_smc_delta").Get("checked").Bool()

	bedProbe = doc.Call("getElementById", "k3d_smc_g29").Get("checked").Bool()

	// Параметры филамента

	docHotTemp, err := parseInputToInt(doc.Call("getElementById", "k3d_smc_hotendTemperature").Get("value").String())
	if err != nil {
		curErr, hasErr = lang.Call("getString", "error.hotend_temp.format").String(), true
	} else if docHotTemp < 150 || docHotTemp > 350 {
		curErr, hasErr = lang.Call("getString", "error.hotend_temp.value").String(), true
	} else {
		hotendTemperature = docHotTemp
	}

	setErrorDescription(doc, lang, "table.hotend_temp.description", curErr, hasErr, allowModify)
	if hasErr {
		errorString = errorString + curErr + "\n"
		hasErr = false
		retErr = true
	}

	docBedTemp, err := parseInputToInt(doc.Call("getElementById", "k3d_smc_bedTemperature").Get("value").String())
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

	docCooling, err := parseInputToInt(doc.Call("getElementById", "k3d_smc_cooling").Get("value").String())
	if err != nil {
		curErr, hasErr = lang.Call("getString", "error.cooling.format").String(), true
	} else {
		docCooling = int(roundFloat(float64(docCooling)*2.55, 0))
		if docCooling < 0 {
			docCooling = 0
		} else if docCooling > 255 {
			docCooling = 255
		}
		cooling = docCooling
	}

	setErrorDescription(doc, lang, "table.cooling.description", curErr, hasErr, allowModify)
	if hasErr {
		errorString = errorString + curErr + "\n"
		hasErr = false
		retErr = true
	}

	docFlow, err := parseInputToInt(doc.Call("getElementById", "k3d_smc_flow").Get("value").String())
	if err != nil {
		curErr, hasErr = lang.Call("getString", "error.flow.format").String(), true
	} else if docFlow < 50 || docFlow > 150 {
		curErr, hasErr = lang.Call("getString", "error.flow.value").String(), true
	} else {
		flow = docFlow
	}

	setErrorDescription(doc, lang, "table.flow.description", curErr, hasErr, allowModify)
	if hasErr {
		errorString = errorString + curErr + "\n"
		hasErr = false
		retErr = true
	}

	docLA, err := parseInputToFloat(doc.Call("getElementById", "k3d_smc_la").Get("value").String())
	if err != nil {
		curErr, hasErr = lang.Call("getString", "error.la.format").String(), true
	} else if docLA < 0 || docLA > 2.0 {
		curErr, hasErr = lang.Call("getString", "error.la.value").String(), true
	} else {
		la = docLA
	}

	setErrorDescription(doc, lang, "table.la.description", curErr, hasErr, allowModify)
	if hasErr {
		errorString += curErr + "\n"
		hasErr = false
		retErr = true
	}

	// Параметры первого слоя

	docFirstLayerLineWidth, err := parseInputToFloat(doc.Call("getElementById", "k3d_smc_firstLayerLineWidth").Get("value").String())
	if err != nil {
		curErr, hasErr = lang.Call("getString", "error.first_layer_line_width.format").String(), true
	} else if docFirstLayerLineWidth < 0.1 || docFirstLayerLineWidth > 2.0 {
		curErr, hasErr = lang.Call("getString", "error.first_layer_line_width.value").String(), true
	} else {
		firstLayerLineWidth = docFirstLayerLineWidth
	}

	setErrorDescription(doc, lang, "table.first_layer_line_width.description", curErr, hasErr, allowModify)
	if hasErr {
		errorString = errorString + curErr + "\n"
		hasErr = false
		retErr = true
	}

	docFirstLayerSpeed, err := parseInputToInt(doc.Call("getElementById", "k3d_smc_firstLayerSpeed").Get("value").String())
	if err != nil {
		curErr, hasErr = lang.Call("getString", "error.first_layer_print_speed.format").String(), true
	} else if docFirstLayerSpeed < 10 || docFirstLayerSpeed > 1000 {
		curErr, hasErr = lang.Call("getString", "error.first_layer_print_speed.value").String(), true
	} else {
		firstLayerSpeed = docFirstLayerSpeed
	}

	setErrorDescription(doc, lang, "table.first_layer_print_speed.description", curErr, hasErr, allowModify)
	if hasErr {
		errorString = errorString + curErr + "\n"
		hasErr = false
		retErr = true
	}

	docZOffset, err := parseInputToFloat(doc.Call("getElementById", "k3d_smc_zOffset").Get("value").String())
	if err != nil {
		curErr, hasErr = lang.Call("getString", "error.z_offset.format").String(), true
	} else if docZOffset < -0.5 || docZOffset > 0.5 {
		curErr, hasErr = lang.Call("getString", "error.z_offset.value").String(), true
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

	docLineWidth, err := parseInputToFloat(doc.Call("getElementById", "k3d_smc_lineWidth").Get("value").String())
	if err != nil {
		curErr, hasErr = lang.Call("getString", "error.line_width.format").String(), true
	} else if docLineWidth < 0.1 || docLineWidth > 2.0 {
		curErr, hasErr = lang.Call("getString", "error.line_width.value").String(), true
	} else {
		lineWidth = docLineWidth
	}

	setErrorDescription(doc, lang, "table.line_width.description", curErr, hasErr, allowModify)
	if hasErr {
		errorString = errorString + curErr + "\n"
		hasErr = false
		retErr = true
	}

	docLayerHeight, err := parseInputToFloat(doc.Call("getElementById", "k3d_smc_layerHeight").Get("value").String())
	if err != nil {
		curErr, hasErr = lang.Call("getString", "error.layer_height.format").String(), true
	} else if docLayerHeight < 0.05 || docLayerHeight > 1.2 {
		curErr, hasErr = lang.Call("getString", "error.layer_height.value").String(), true
	} else {
		layerHeight = docLayerHeight
	}

	setErrorDescription(doc, lang, "table.layer_height.description", curErr, hasErr, allowModify)
	if hasErr {
		errorString = errorString + curErr + "\n"
		hasErr = false
		retErr = true
	}

	docPrintSpeed, err := parseInputToInt(doc.Call("getElementById", "k3d_smc_printSpeed").Get("value").String())
	if err != nil {
		curErr, hasErr = lang.Call("getString", "error.print_speed.format").String(), true
	} else if docPrintSpeed < 60 || docPrintSpeed > 300 {
		curErr, hasErr = lang.Call("getString", "error.print_speed.value").String(), true
	} else {
		printSpeed = docPrintSpeed
	}

	setErrorDescription(doc, lang, "table.print_speed.description", curErr, hasErr, allowModify)
	if hasErr {
		errorString = errorString + curErr + "\n"
		hasErr = false
		retErr = true
	}

	docSlowAcceleration, err := parseInputToInt(doc.Call("getElementById", "k3d_smc_slowAcceleration").Get("value").String())
	if err != nil {
		curErr, hasErr = lang.Call("getString", "error.slow_acceleration.format").String(), true
	} else if docSlowAcceleration < 100 || docSlowAcceleration > 50000 {
		curErr, hasErr = lang.Call("getString", "error.slow_acceleration.value").String(), true
	} else {
		slowAcceleration = docSlowAcceleration
	}

	setErrorDescription(doc, lang, "table.slow_acceleration.description", curErr, hasErr, allowModify)
	if hasErr {
		errorString = errorString + curErr + "\n"
		hasErr = false
		retErr = true
	}

	// Параметры калибровки

	docStartAcceleration, err := parseInputToInt(doc.Call("getElementById", "k3d_smc_startAcceleration").Get("value").String())
	if err != nil {
		curErr, hasErr = lang.Call("getString", "error.start_acceleration.format").String(), true
	} else if docStartAcceleration < 100 || docStartAcceleration > 50000 {
		curErr, hasErr = lang.Call("getString", "error.start_acceleration.value").String(), true
	} else {
		startAcceleration = docStartAcceleration
	}

	setErrorDescription(doc, lang, "table.start_acceleration.description", curErr, hasErr, allowModify)
	if hasErr {
		errorString = errorString + curErr + "\n"
		hasErr = false
		retErr = true
	}

	docEndAcceleration, err := parseInputToInt(doc.Call("getElementById", "k3d_smc_endAcceleration").Get("value").String())
	if err != nil {
		curErr, hasErr = lang.Call("getString", "error.end_acceleration.format").String(), true
	} else if docEndAcceleration < 100 || docEndAcceleration > 50000 {
		curErr, hasErr = lang.Call("getString", "error.end_acceleration.value").String(), true
	} else {
		endAcceleration = docEndAcceleration
	}

	setErrorDescription(doc, lang, "table.end_acceleration.description", curErr, hasErr, allowModify)
	if hasErr {
		errorString = errorString + curErr + "\n"
		hasErr = false
		retErr = true
	}

	docNumSegment, err := parseInputToInt(doc.Call("getElementById", "k3d_smc_numSegments").Get("value").String())
	if err != nil {
		curErr, hasErr = lang.Call("getString", "error.num_segments.format").String(), true
	} else if docNumSegment < 2 || docNumSegment > 100 {
		curErr, hasErr = lang.Call("getString", "error.num_segments.value").String(), true
	} else {
		numSegments = docNumSegment
	}

	setErrorDescription(doc, lang, "table.num_segments.description", curErr, hasErr, allowModify)
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

		deltaAcceleration := math.Abs((float64(endAcceleration-startAcceleration) / float64(numSegments-1)))

		// generate calibration parameters
		caliParams := ""
		for i := 0; i < numSegments; i++ {
			caliParams += fmt.Sprintf(segmentStr, numSegments-i, fmt.Sprint(roundFloat(float64(endAcceleration)-deltaAcceleration*float64(i), 3)))
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
	deltaAcceleration := math.Abs((float64(endAcceleration-startAcceleration) / float64(numSegments-1)))
	caliParams += "; ====================\n; Поддержите выход новых калибраторов, инструкций и видео!\n;https://donate.stream/dmitrysorkin\n; ====================\n"
	for i := 0; i < numSegments; i++ {
		caliParams += fmt.Sprintf(segmentStr, numSegments-i, fmt.Sprint(roundFloat(float64(endAcceleration)-deltaAcceleration*float64(i), 3)))
	}

	fileName := fmt.Sprintf("K3D_SMC_H%d-B%d_%d-%d_d%s.gcode", hotendTemperature, bedTemperature, startAcceleration, endAcceleration, int(roundFloat(deltaAcceleration, 2)))
	js.Global().Call("beginSaveFile", fileName)

	// gcode initialization
	write("; generated by K3D max acceleration for IS ", js.Global().Get("calibrator_version").String(), "\n",
		"; Written by Dmitry Sorkin @ http://k3d.tech/, Kekht and YTKAB0BP\n",
		fmt.Sprintf(";Bedsize: %s:%s [mm]\n", fmt.Sprint(roundFloat(bedX, 1)), fmt.Sprint(roundFloat(bedY, 1))),
		fmt.Sprintf(";Firmware (0-Marlin, 1-Klipper, 2-RRF): %d\n", firmware),
		fmt.Sprintf(";Z-offset: %s [mm]\n", fmt.Sprint(roundFloat(zOffset, 3))),
		fmt.Sprintf(";Delta: %s\n", strconv.FormatBool(delta)),
		fmt.Sprintf(";G29: %s\n", strconv.FormatBool(bedProbe)),
		fmt.Sprintf(";Temp: %d/%d [°C]\n", hotendTemperature, bedTemperature),
		fmt.Sprintf(";Flow: %d\n", flow),
		fmt.Sprintf(";Fan: %s\n", fmt.Sprint(roundFloat(float64(cooling)/2.55, 0))),
		fmt.Sprintf(";LA/PA: %s\n", fmt.Sprint(roundFloat(la, 3))),
		fmt.Sprintf(";First layer speed: %d [mm/s]\n", firstLayerSpeed),
		fmt.Sprintf(";First layer line width: %s [mm]\n", fmt.Sprint(roundFloat(lineWidth, 2))),
		fmt.Sprintf(";Line width: %s [mm]\n", fmt.Sprint(roundFloat(lineWidth, 2))),
		fmt.Sprintf(";Layer height: %s [mm]\n", fmt.Sprint(roundFloat(layerHeight, 2))),
		fmt.Sprintf(";Print speed: %d [mm/s]\n", printSpeed),
		caliParams)

	var g29str string
	if bedProbe {
		g29str = "G29"
	} else {
		g29str = ""
	}
	replacer := strings.NewReplacer("$BEDTEMP", strconv.Itoa(bedTemperature),
		"$HOTTEMP", strconv.Itoa(hotendTemperature),
		"$G29", g29str,
		"$FLOW", strconv.Itoa(flow))
	write(replacer.Replace(startGcode), "\n")

	write("M82\n", "M106 S0\n")

	// set acceleration limits so calibrator may work
	maxAcceleration := int(math.Max(float64(startAcceleration), float64(endAcceleration)))
	if firmware == 0 || firmware == 2 {
		// Marlin or RRF
		write(fmt.Sprintf("M201 X%d Y%d\n", maxAcceleration, maxAcceleration))
	} else if firmware == 1 {
		// Klipper
		write(fmt.Sprintf("SET_VELOCITY_LIMIT ACCEL=%d ACCEL_TO_DECEL=%d\n", maxAcceleration, maxAcceleration))
	}

	// set slow acceleration for 1st layer
	write(generateAccelerationCommand(slowAcceleration))

	// generate first layer
	var bedCenter Point
	if delta {
		bedCenter.X, bedCenter.Y, bedCenter.Z = 0, 0, layerHeight
	} else {
		bedCenter.X, bedCenter.Y, bedCenter.Z = bedX/2, bedY/2, layerHeight
	}
	currentE = 0
	currentSpeed = firstLayerSpeed
	currentCoordinates.X, currentCoordinates.Y, currentCoordinates.Z = 0, 0, 0

	// move to layer height to avoid nozzle collizion with bed
	write(fmt.Sprintf("G1 Z%s\n", fmt.Sprint(roundFloat(layerHeight+zOffset, 2))))

	// make printer think, that he is on layerHeight
	write(fmt.Sprintf("G92 Z%s\n", fmt.Sprint(roundFloat(layerHeight, 2))))
	currentCoordinates.Z = layerHeight

	// calculate model parameters
	stepWidth := 2 * minLineLength * math.Sin(angle/2)
	modelSizeX := stepWidth * float64(numSegments)
	modelSizeY := minLineLength * math.Cos(angle/2)

	// purge nozzle
	var purgeStart Point
	purgeStart.X, purgeStart.Y, purgeStart.Z = 15.0, bedCenter.Y-modelSizeY/2-10.0, currentCoordinates.Z
	purgeTwo := purgeStart
	purgeTwo.X = bedCenter.X + bedX/2 - 15.0
	purgeThree := purgeTwo
	purgeThree.Y += firstLayerLineWidth
	purgeEnd := purgeThree
	purgeEnd.X = purgeStart.X

	// move to start of purge
	write(generateMove(currentCoordinates, purgeStart, 0.0, travelSpeed))

	// add purge to gcode
	write(generateMove(currentCoordinates, purgeTwo, firstLayerLineWidth, firstLayerSpeed))
	write(generateMove(currentCoordinates, purgeThree, firstLayerLineWidth, firstLayerSpeed))
	write(generateMove(currentCoordinates, purgeEnd, firstLayerLineWidth, firstLayerSpeed))

	// generate raft trajectory
	// calc raft size
	raftSizeX := modelSizeX + lineWidth/2
	raftSizeY := modelSizeY + lineWidth/2
	// move to raft start
	var point0 Point
	point0.X, point0.Y, point0.Z = bedCenter.X-raftSizeX/2, bedCenter.Y+raftSizeY/2, layerHeight
	write(generateRetraction())
	write(generateMove(currentCoordinates, point0, 0.0, travelSpeed))
	write(generateDeretraction())
	// generate raft gcode
	curX := point0.X
	for curX <= bedCenter.X+raftSizeX/2 {
		write(generateRelativeMove(0.0, -raftSizeY, 0.0, firstLayerLineWidth, firstLayerSpeed))
		write(generateRelativeMove(firstLayerLineWidth, 0.0, 0.0, firstLayerLineWidth, firstLayerSpeed))
		write(generateRelativeMove(0.0, raftSizeY, 0.0, firstLayerLineWidth, firstLayerSpeed))
		write(generateRelativeMove(firstLayerLineWidth, 0.0, 0.0, firstLayerLineWidth, firstLayerSpeed))
		curX += 2 * firstLayerLineWidth
	}

	// calculate model start poing coordinates
	var modelStartPoint Point
	modelStartPoint.X, modelStartPoint.Y, modelStartPoint.Z = bedCenter.X-modelSizeX/2, bedCenter.Y+modelSizeY/2, layerHeight

	// model generation
	for currentCoordinates.Z <= modelHeight {
		// modify model start point Z coordinate
		modelStartPoint.Z += layerHeight

		// move to start of calibration model
		if currentCoordinates.Z == layerHeight*2 {
			// generate retraction only when moving from raft to model
			write(generateRetraction())
			write(generateMove(currentCoordinates, modelStartPoint, 0.0, travelSpeed))
			write(generateDeretraction())
		} else {
			// else move to next layer start point without retraction
			write(generateMove(currentCoordinates, modelStartPoint, 0.0, travelSpeed))
		}

		// generate upper model perimeters
		for curSegment := 0; curSegment < numSegments; curSegment++ {
			// set segment acceleration
			write(generateAccelerationCommand(startAcceleration + curSegment*int(deltaAcceleration)))
			// m
			write(generateRelativeMove(stepWidth/2, -modelSizeY, 0.0, lineWidth, printSpeed))
			write(generateRelativeMove(stepWidth/2, modelSizeY, 0.0, lineWidth, printSpeed))
		}
		// set slow acceleration for lower perimeter
		write(generateAccelerationCommand(slowAcceleration))

		// move to lower perimeters
		write(generateRelativeMove(0.0, -spacing/math.Sin(angle/2), 0.0, lineWidth, printSpeed))
		for curSegment := numSegments; curSegment >= 0; curSegment-- {
			write(generateRelativeMove(-stepWidth/2, -modelSizeY, 0.0, lineWidth, printSpeed))
			write(generateRelativeMove(-stepWidth/2, modelSizeY, 0.0, lineWidth, printSpeed))
		}

		// print last line
		write(generateRelativeMove(0.0, spacing/math.Sin(angle/2), 0.0, lineWidth, printSpeed))
	}

	// end gcode
	write(endGcode)

	// write calibration parameters to resultContainer
	js.Global().Call("showError", caliParams)

	// save file
	js.Global().Call("finishFile")

	return js.ValueOf(nil)
}

func generateAccelerationCommand(acc int) string {
	if firmware == 0 || firmware == 1 {
		// same command for marlin and klipper
		return fmt.Sprintf("M204 S%d\n", acc)
	} else if firmware == 2 {
		return fmt.Sprintf("M204 P%d T%d\n", acc, acc)
	}

	return ";no firmware information\n"
}

func generateLACommand(kFactor float64) string {
	if firmware == 0 {
		return fmt.Sprintf("M900 K%s\n", fmt.Sprint(roundFloat(kFactor, 3)))
	} else if firmware == 1 {
		return fmt.Sprintf("SET_PRESSURE_ADVANCE ADVANCE=%s\n", fmt.Sprint(roundFloat(kFactor, 3)))
	} else if firmware == 2 {
		return fmt.Sprintf("M572 D0 S%s\n", fmt.Sprint(roundFloat(kFactor, 3)))
	}

	return ";no firmware information\n"
}

func generateRelativeMove(x, y, z, width float64, speed int) string {
	endPoint := currentCoordinates
	endPoint.X += x
	endPoint.Y += y
	endPoint.Z += z
	return generateMove(currentCoordinates, endPoint, width, speed)
}

func generateMove(start, end Point, width float64, speed int) string {

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
	command += "\n"
	currentCoordinates = end

	return command
}

func calcExtrusion(start, end Point, width float64) float64 {
	lineLength := math.Sqrt(float64(math.Pow((end.X-start.X), 2) + math.Pow((end.Y-start.Y), 2)))
	extrusion := width * layerHeight * lineLength * 4 / math.Pi / math.Pow(filamentDiameter, 2)
	return extrusion
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
