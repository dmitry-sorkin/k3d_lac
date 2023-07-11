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
	bedX, bedY, zOffset, retractLength, firstLayerLineWidth, lineWidth, layerHeight, initKFactor, endKFactor, segmentHeight                                                 float64
	firmware, travelSpeed, hotendTemperature, bedTemperature, retractSpeed, cooling, firstLayerPrintSpeed, fastPrintSpeed, slowPrintSpeed, numSegments, numPerimeters, flow int
	g29, retracted, delta                                                                                                                                                   bool
	// Current variables
	currentCoordinates Point
	currentSpeed       int
	currentE           float64
)

const caliVersion = "v1.2"

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
}

func check() bool {
	errorString := ""
	doc := js.Global().Get("document")
	doc.Call("getElementById", "resultContainer").Set("innerHTML", "")

	// Параметры принтера

	docBedX, err := parseInputToFloat(doc.Call("getElementById", "k3d_la_bedX").Get("value").String())
	if err != nil {
		errorString += "Ошибка формата: размер оси X\n"
	} else if docBedX < 100 || docBedX > 1000 {
		errorString += "Ошибка значения: размер стола по оси X должен быть от 100 до 1000 мм\n"
	} else {
		bedX = docBedX
	}

	docBedY, err := parseInputToFloat(doc.Call("getElementById", "k3d_la_bedY").Get("value").String())
	if err != nil {
		errorString += "Ошибка формата: размер оси Y\n"
	} else if docBedY < 100 || docBedY > 1000 {
		errorString += "Ошибка значения: размер стола по оси Y должен быть от 100 до 1000 мм\n"
	} else {
		bedY = docBedY
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
		errorString += "Ошибка формата: не выбрана прошивка\n"
	}

	delta = doc.Call("getElementById", "k3d_la_delta").Get("checked").Bool()

	g29 = doc.Call("getElementById", "k3d_la_g29").Get("checked").Bool()

	docTravelSpeed, err := parseInputToInt(doc.Call("getElementById", "k3d_la_travelSpeed").Get("value").String())
	if err != nil {
		errorString += "Ошибка формата: скорость перемещений\n"
	} else if docTravelSpeed < 10 || docTravelSpeed > 1000 {
		errorString += "Ошибка значения: скорость перемещений должна быть от 10 до 1000 мм/с\n"
	} else {
		travelSpeed = docTravelSpeed
	}

	// Параметры филамента

	docHotTemp, err := parseInputToInt(doc.Call("getElementById", "k3d_la_hotendTemperature").Get("value").String())
	if err != nil {
		errorString += "Ошибка формата: температура хотэнда\n"
	} else if docHotTemp < 150 || docHotTemp > 350 {
		errorString += "Ошибка значения: температура хотэнда должна быть от 150 до 350 градусов\n"
	} else {
		hotendTemperature = docHotTemp
	}

	docBedTemp, err := parseInputToInt(doc.Call("getElementById", "k3d_la_bedTemperature").Get("value").String())
	if err != nil {
		errorString += "Ошибка формата: температура стола\n"
	} else if docBedTemp > 150 {
		errorString += "Ошибка значения: температура стола должна быть от 0 до 150 градусов\n"
	} else {
		bedTemperature = docBedTemp
	}

	docCooling, err := parseInputToInt(doc.Call("getElementById", "k3d_la_cooling").Get("value").String())
	if err != nil {
		errorString += "Ошибка формата: скорость вентилятора\n"
	} else if docCooling < 0 || docCooling > 100 {
		errorString += "Ошибка значения: скорость вентилятора должна быть от 0 до 100%\n"
	} else {
		cooling = int(roundFloat(float64(docCooling)*2.55, 0))
	}

	docRetractLength, err := parseInputToFloat(doc.Call("getElementById", "k3d_la_retractLength").Get("value").String())
	if err != nil {
		errorString += "Ошибка формата: длина отката\n"
	} else if docRetractLength < 0.1 || docRetractLength > 20 {
		errorString += "Ошибка значения: длина отката должна быть от 0.1 до 20 мм\n"
	} else {
		retractLength = docRetractLength
	}

	docRetractSpeed, err := parseInputToInt(doc.Call("getElementById", "k3d_la_retractSpeed").Get("value").String())
	if err != nil {
		errorString += "Ошибка формата: скорость отката\n"
	} else if docRetractSpeed < 5 || docRetractSpeed > 150 {
		errorString += "Ошибка значения: скорость отката должна быть от 5 до 150 мм/с\n"
	} else {
		retractSpeed = docRetractSpeed
	}

	docFlow, err := parseInputToInt(doc.Call("getElementById", "k3d_la_flow").Get("value").String())
	if err != nil {
		errorString += "Ошибка формата: поток\n"
	} else if docFlow < 50 || docFlow > 150 {
		errorString += "Ошибка значения: поток должен быть от 50 до 150%\n"
	} else {
		flow = docFlow
	}

	// Параметры первого слоя

	docFirstLayerLineWidth, err := parseInputToFloat(doc.Call("getElementById", "k3d_la_firstLayerLineWidth").Get("value").String())
	if err != nil {
		errorString += "Ошибка формата: ширина линии первого слоя\n"
	} else if docFirstLayerLineWidth < 0.1 || docFirstLayerLineWidth > 2.0 {
		errorString += "Ошибка значения: ширина линии первого слоя должна быть от 0.1 до 2.0 мм\n"
	} else {
		firstLayerLineWidth = docFirstLayerLineWidth
	}

	docFirstLayerPrintSpeed, err := parseInputToInt(doc.Call("getElementById", "k3d_la_firstLayerSpeed").Get("value").String())
	if err != nil {
		errorString += "Ошибка формата: скорость печати первого слоя\n"
	} else if docFirstLayerPrintSpeed < 10 || docFirstLayerPrintSpeed > 1000 {
		errorString += "Ошибка значения: скорость печати первого слоя должна быть от 10 до 1000 мм/с\n"
	} else {
		firstLayerPrintSpeed = docFirstLayerPrintSpeed
	}

	docZOffset, err := parseInputToFloat(doc.Call("getElementById", "k3d_la_zOffset").Get("value").String())
	if err != nil {
		errorString += "Ошибка формата: Z-Offset\n"
	} else if docZOffset < -0.5 || docZOffset > 0.5 {
		errorString += "Ошибка значения: Z-Offset должен быть от -0.5 до 0.5 мм\n"
	} else {
		zOffset = docZOffset
	}

	// Параметры модели

	docNumPerimeters, err := parseInputToInt(doc.Call("getElementById", "k3d_la_numPerimeters").Get("value").String())
	if err != nil {
		errorString += "Ошибка формата: количество периметров\n"
	} else if docNumPerimeters < 1 || docNumPerimeters > 5 {
		errorString += "Ошибка значения: количество периметров должно быть от 1 до 5\n"
	} else {
		numPerimeters = docNumPerimeters
	}

	docLineWidth, err := parseInputToFloat(doc.Call("getElementById", "k3d_la_lineWidth").Get("value").String())
	if err != nil {
		errorString += "Ошибка формата: ширина линии\n"
	} else if docLineWidth < 0.1 || docLineWidth > 2.0 {
		errorString += "Ошибка значения: ширина линии должна быть от 0.1 до 2.0 мм\n"
	} else {
		lineWidth = docLineWidth
	}

	docLayerHeight, err := parseInputToFloat(doc.Call("getElementById", "k3d_la_layerHeight").Get("value").String())
	if err != nil {
		errorString += "Ошибка формата: толщина слоя\n"
	} else if docLayerHeight < 0.05 || docLayerHeight > 1.2 {
		errorString += "Ошибка значения: толщина слоя должна быть от 0.05 до 1.2 мм\n"
	} else {
		layerHeight = docLayerHeight
	}

	docFastPrintSpeed, err := parseInputToInt(doc.Call("getElementById", "k3d_la_fastPrintSpeed").Get("value").String())
	if err != nil {
		errorString += "Ошибка формата: скорость печати быстрых участков\n"
	} else if docFastPrintSpeed < 10 || docFastPrintSpeed > 1000 {
		errorString += "Ошибка значения: скорость печати быстрых участков должна быть от 10 до 1000 мм/с\n"
	} else {
		fastPrintSpeed = docFastPrintSpeed
	}

	docSlowPrintSpeed, err := parseInputToInt(doc.Call("getElementById", "k3d_la_slowPrintSpeed").Get("value").String())
	if err != nil {
		errorString += "Ошибка формата: скорость печати медленных участков\n"
	} else if docSlowPrintSpeed < 10 || docSlowPrintSpeed > 1000 {
		errorString += "Ошибка значения: скорость печати медленных участков должна быть от 10 до 1000 мм/с\n"
	} else {
		slowPrintSpeed = docSlowPrintSpeed
	}

	// Параметры калибровки

	docInitKFactor, err := parseInputToFloat(doc.Call("getElementById", "k3d_la_initKFactor").Get("value").String())
	if err != nil {
		errorString += "Ошибка формата: начальное значение коэффициента LA\n"
	} else if docInitKFactor < 0.0 || docInitKFactor > 2.0 {
		errorString += "Ошибка значения: начальное значение коэффициента LA должно быть от 0.0 до 2.0\n"
	} else {
		initKFactor = docInitKFactor
	}

	docEndKFactor, err := parseInputToFloat(doc.Call("getElementById", "k3d_la_endKFactor").Get("value").String())
	if err != nil {
		errorString += "Ошибка формата: конечное значение коэффициента LA\n"
	} else if docEndKFactor < 0.0 || docEndKFactor > 2.0 {
		errorString += "Ошибка значения: конечное значение коэффициента LA должно быть от 0.0 до 2.0\n"
	} else {
		endKFactor = docEndKFactor
	}

	docNumSegment, err := parseInputToInt(doc.Call("getElementById", "k3d_la_numSegments").Get("value").String())
	if err != nil {
		errorString += "Ошибка формата: количество сегментов\n"
	} else if docNumSegment < 2 || docNumSegment > 100 {
		errorString += "Ошибка значения: количество сегментов должно быть от 2 до 100\n"
	} else {
		numSegments = docNumSegment
	}

	docSegmentHeight, err := parseInputToFloat(doc.Call("getElementById", "k3d_la_segmentHeight").Get("value").String())
	if err != nil {
		errorString += "Ошибка формата: высота сегмента\n"
	} else if docSegmentHeight < 0.5 || docSegmentHeight > 10.0 {
		errorString += "Ошибка значения: высота сегмента должна быть от 0.5 до 10.0 мм\n"
	} else {
		segmentHeight = docSegmentHeight
	}

	// end check of parameters
	if errorString == "" {
		println("OK")
		return true
	} else {
		println(errorString)
		js.Global().Call("showError", errorString)
		return false
	}
}

func generate(this js.Value, i []js.Value) interface{} {
	// check and initialize variables
	if !check() {
		return js.ValueOf(nil)
	}

	// generate calibration parameters
	caliParams := ""
	deltaKFactor := math.Abs((endKFactor - initKFactor) / float64(numSegments-1))
	maxKFactor := math.Max(initKFactor, endKFactor)
	minKFactor := math.Min(initKFactor, endKFactor)
	currentKFactor := minKFactor
	caliParams += "; ====================\n; Поддержите выход новых калибраторов, инструкций и видео!\n; Пожертвование из РФ: https://donate.stream/dmitrysorkin\n; Пожертвование из-за рубежа: https://www.donationalerts.com/r/dsorkin\n; ====================\n"
	for i := 0; i < numSegments; i++ {
		caliParams += fmt.Sprintf("; Segment:%d K-Factor:%s\n", numSegments-i, fmt.Sprint(roundFloat(maxKFactor-deltaKFactor*float64(i), 3)))
	}

	// gcode initialization
	gcode := make([]string, 0, 1)
	gcode = append(gcode, "; generated by K3D LA calibration ",
		caliVersion, "\n",
		"; Written by Dmitry Sorkin @ http://k3d.tech/\n",
		"; and Kekht\n",
		caliParams,
		fmt.Sprintf("; Bedsize: %s:%s\n", fmt.Sprint(roundFloat(bedX, 0)), fmt.Sprint(roundFloat(bedY, 0))),
		fmt.Sprintf("; Temperature H:%d B:%d °C\n", hotendTemperature, bedTemperature),
		fmt.Sprintf("; Line width: %s-%s mm\n", fmt.Sprint(roundFloat(lineWidth, 2)), fmt.Sprint(roundFloat(firstLayerLineWidth, 2))),
		fmt.Sprintf("; Layer height: %s mm\n", fmt.Sprint(roundFloat(layerHeight, 2))),
		fmt.Sprintf("; Segments: %dx%s mm\n", numSegments, fmt.Sprint(roundFloat(segmentHeight, 2))),
		fmt.Sprintf("; Print speed: %d, %d, %d mm/s\n", firstLayerPrintSpeed, slowPrintSpeed, fastPrintSpeed),
		fmt.Sprintf("; Retractions: %smm @ %d mm/s\n", fmt.Sprint(roundFloat(retractLength, 2)), retractSpeed),
		"M104 S150\n",
		fmt.Sprintf("M190 S%d\n", bedTemperature),
		fmt.Sprintf("M109 S%d\n", hotendTemperature),
		generateLACommand(currentKFactor),
		"G28\n")
	if g29 {
		gcode = append(gcode, "G29\n")
	}
	gcode = append(gcode, "G92 E0\n",
		"G90\n",
		"M82\n",
		"M106 S0\n",
		fmt.Sprintf("M221 S%d\n", flow))

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
	gcode = append(gcode, fmt.Sprintf("G1 Z%s\n", fmt.Sprint(roundFloat(layerHeight+zOffset, 2))))

	// make printer think, that he is on layerHeight
	gcode = append(gcode, fmt.Sprintf("G92 Z%s\n", fmt.Sprint(roundFloat(layerHeight, 2))))
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
	gcode = append(gcode, generateMove(currentCoordinates, purgeStart, 0.0, travelSpeed)...)

	// add purge to gcode
	gcode = append(gcode, generateMove(currentCoordinates, purgeTwo, firstLayerLineWidth, firstLayerPrintSpeed)...)
	gcode = append(gcode, generateMove(currentCoordinates, purgeThree, firstLayerLineWidth, firstLayerPrintSpeed)...)
	gcode = append(gcode, generateMove(currentCoordinates, purgeEnd, firstLayerLineWidth, firstLayerPrintSpeed)...)

	// generate raft trajectory
	trajectory := generateZigZagTrajectory(bedCenter, firstLayerLineWidth, modelWidth+10.0)

	// move to start of raft
	gcode = append(gcode, generateRetraction())
	gcode = append(gcode, generateMove(currentCoordinates, trajectory[0], 0.0, travelSpeed)...)
	gcode = append(gcode, generateDeretraction())

	// print raft
	for i := 1; i < len(trajectory); i++ {
		gcode = append(gcode, generateMove(currentCoordinates, trajectory[i], firstLayerLineWidth, firstLayerPrintSpeed)...)
	}

	// generate model
	layersPerSegment := int(segmentHeight / layerHeight)
	for i := 1; i < numSegments*layersPerSegment; i++ {
		// add layer start comment
		gcode = append(gcode, fmt.Sprintf(";layer #%s\n", fmt.Sprint(roundFloat(currentCoordinates.Z/layerHeight, 0))))

		// change fan speed
		if i < 4 {
			gcode = append(gcode, fmt.Sprintf("M106 S%s\n", fmt.Sprint(roundFloat(float64(cooling*i/3), 0))))
		}

		// modify print settings if switching segments
		addition := 0.0
		if i%layersPerSegment == 0 {
			currentKFactor += deltaKFactor
			gcode = append(gcode, generateLACommand(currentKFactor))
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
		gcode = append(gcode, generateMove(currentCoordinates, layerStart, 0.0, travelSpeed)...)
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
			gcode = append(gcode, generateRelativeMove(currentModelWidth/2, 0, 0, lineWidth, fastPrintSpeed)...)
			// print right line
			gcode = append(gcode, generateRelativeMove(0, -rightLongLine, 0, lineWidth, fastPrintSpeed)...)
			gcode = append(gcode, generateRelativeMove(0, -rightShortLine, 0, lineWidth, slowPrintSpeed)...)
			gcode = append(gcode, generateRelativeMove(0, -rightLongLine, 0, lineWidth, fastPrintSpeed)...)
			// print front line
			gcode = append(gcode, generateRelativeMove(-frontLongLine, 0, 0, lineWidth, fastPrintSpeed)...)
			gcode = append(gcode, generateRelativeMove(-frontShortLine, 0, 0, lineWidth, slowPrintSpeed)...)
			gcode = append(gcode, generateRelativeMove(-frontLongLine, 0, 0, lineWidth, fastPrintSpeed)...)
			// print left line
			gcode = append(gcode, generateRelativeMove(0, leftLongLine, 0, lineWidth, fastPrintSpeed)...)
			gcode = append(gcode, generateRelativeMove(0, leftShortLine, 0, lineWidth, slowPrintSpeed)...)
			gcode = append(gcode, generateRelativeMove(0, leftLongLine, 0, lineWidth, fastPrintSpeed)...)
			// print back line left part
			gcode = append(gcode, generateRelativeMove(currentModelWidth/2, 0, 0, lineWidth, fastPrintSpeed)...)
			// move to start of next perimeter if it exists
			if j != numPerimeters-1 {
				gcode = append(gcode, generateRelativeMove(0, -lineWidth, 0, 0.0, fastPrintSpeed)...)
			}
		}
	}

	// end gcode
	gcode = append(gcode, ";end gcode\n",
		"M104 S0\n",
		"M140 S0\n",
		"M106 S0\n",
		fmt.Sprintf("G1 Z%f F600\n", currentCoordinates.Z+5),
		"M84")

	outputGCode := ""
	for i := 0; i < len(gcode); i++ {
		outputGCode += gcode[i]
	}

	// write calibration parameters to resultContainer
	js.Global().Call("showError", caliParams)

	// save file
	fileName := fmt.Sprintf("K3D_LA_H%d-B%d_%s-%s_d%s.gcode", hotendTemperature, bedTemperature, fmt.Sprint(roundFloat(initKFactor, 2)), fmt.Sprint(roundFloat(endKFactor, 2)), fmt.Sprint(roundFloat(deltaKFactor, 3)))
	js.Global().Call("saveTextAsFile", fileName, outputGCode)

	return js.ValueOf(nil)
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
