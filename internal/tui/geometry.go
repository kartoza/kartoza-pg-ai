package tui

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"strings"
)

// Geometry types for rendering
type Point struct {
	X, Y float64
}

type LineString struct {
	Points []Point
}

type Polygon struct {
	Rings [][]Point // First ring is exterior, rest are holes
}

type MultiPoint struct {
	Points []Point
}

type MultiLineString struct {
	Lines []LineString
}

type MultiPolygon struct {
	Polygons []Polygon
}

type GeometryCollection struct {
	Geometries []interface{}
}

// GeometryRenderer renders geometries to PNG images
type GeometryRenderer struct {
	Width      int
	Height     int
	Padding    int
	Background color.Color
	LineColor  color.Color
	FillColor  color.Color
	PointColor color.Color
}

// NewGeometryRenderer creates a new renderer with default settings
func NewGeometryRenderer(width, height int) *GeometryRenderer {
	return &GeometryRenderer{
		Width:      width,
		Height:     height,
		Padding:    10,
		Background: color.RGBA{30, 30, 30, 255},     // Dark gray
		LineColor:  color.RGBA{255, 165, 0, 255},    // Orange
		FillColor:  color.RGBA{255, 165, 0, 80},     // Semi-transparent orange
		PointColor: color.RGBA{0, 200, 255, 255},    // Cyan
	}
}

// RenderGeometries renders a slice of geometry values to a PNG image
func (r *GeometryRenderer) RenderGeometries(geomValues []string) ([]byte, error) {
	// Parse all geometries
	var allGeoms []interface{}
	for _, val := range geomValues {
		geom, err := parseGeometry(val)
		if err != nil {
			continue // Skip unparseable geometries
		}
		if geom != nil {
			allGeoms = append(allGeoms, geom)
		}
	}

	if len(allGeoms) == 0 {
		return nil, fmt.Errorf("no valid geometries to render")
	}

	// Calculate bounding box
	minX, minY, maxX, maxY := r.calculateBounds(allGeoms)

	// Create image
	img := image.NewRGBA(image.Rect(0, 0, r.Width, r.Height))

	// Fill background
	for y := 0; y < r.Height; y++ {
		for x := 0; x < r.Width; x++ {
			img.Set(x, y, r.Background)
		}
	}

	// Calculate transform
	dataWidth := maxX - minX
	dataHeight := maxY - minY
	if dataWidth == 0 {
		dataWidth = 1
	}
	if dataHeight == 0 {
		dataHeight = 1
	}

	drawWidth := float64(r.Width - 2*r.Padding)
	drawHeight := float64(r.Height - 2*r.Padding)

	scale := math.Min(drawWidth/dataWidth, drawHeight/dataHeight)
	offsetX := float64(r.Padding) + (drawWidth-dataWidth*scale)/2
	offsetY := float64(r.Padding) + (drawHeight-dataHeight*scale)/2

	transform := func(p Point) (int, int) {
		x := int(offsetX + (p.X-minX)*scale)
		y := int(float64(r.Height) - offsetY - (p.Y-minY)*scale) // Flip Y
		return x, y
	}

	// Render all geometries
	for _, geom := range allGeoms {
		r.renderGeometry(img, geom, transform)
	}

	// Encode to PNG
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func (r *GeometryRenderer) calculateBounds(geoms []interface{}) (minX, minY, maxX, maxY float64) {
	minX, minY = math.MaxFloat64, math.MaxFloat64
	maxX, maxY = -math.MaxFloat64, -math.MaxFloat64

	var updateBounds func(p Point)
	updateBounds = func(p Point) {
		if p.X < minX {
			minX = p.X
		}
		if p.X > maxX {
			maxX = p.X
		}
		if p.Y < minY {
			minY = p.Y
		}
		if p.Y > maxY {
			maxY = p.Y
		}
	}

	for _, geom := range geoms {
		switch g := geom.(type) {
		case Point:
			updateBounds(g)
		case LineString:
			for _, p := range g.Points {
				updateBounds(p)
			}
		case Polygon:
			for _, ring := range g.Rings {
				for _, p := range ring {
					updateBounds(p)
				}
			}
		case MultiPoint:
			for _, p := range g.Points {
				updateBounds(p)
			}
		case MultiLineString:
			for _, line := range g.Lines {
				for _, p := range line.Points {
					updateBounds(p)
				}
			}
		case MultiPolygon:
			for _, poly := range g.Polygons {
				for _, ring := range poly.Rings {
					for _, p := range ring {
						updateBounds(p)
					}
				}
			}
		}
	}

	// Add small buffer
	buffer := math.Max(maxX-minX, maxY-minY) * 0.05
	if buffer == 0 {
		buffer = 1
	}
	minX -= buffer
	minY -= buffer
	maxX += buffer
	maxY += buffer

	return
}

func (r *GeometryRenderer) renderGeometry(img *image.RGBA, geom interface{}, transform func(Point) (int, int)) {
	switch g := geom.(type) {
	case Point:
		x, y := transform(g)
		r.drawPoint(img, x, y)
	case LineString:
		r.drawLineString(img, g.Points, transform)
	case Polygon:
		r.drawPolygon(img, g, transform)
	case MultiPoint:
		for _, p := range g.Points {
			x, y := transform(p)
			r.drawPoint(img, x, y)
		}
	case MultiLineString:
		for _, line := range g.Lines {
			r.drawLineString(img, line.Points, transform)
		}
	case MultiPolygon:
		for _, poly := range g.Polygons {
			r.drawPolygon(img, poly, transform)
		}
	}
}

func (r *GeometryRenderer) drawPoint(img *image.RGBA, x, y int) {
	radius := 3
	for dy := -radius; dy <= radius; dy++ {
		for dx := -radius; dx <= radius; dx++ {
			if dx*dx+dy*dy <= radius*radius {
				px, py := x+dx, y+dy
				if px >= 0 && px < r.Width && py >= 0 && py < r.Height {
					img.Set(px, py, r.PointColor)
				}
			}
		}
	}
}

func (r *GeometryRenderer) drawLineString(img *image.RGBA, points []Point, transform func(Point) (int, int)) {
	if len(points) < 2 {
		return
	}

	for i := 0; i < len(points)-1; i++ {
		x1, y1 := transform(points[i])
		x2, y2 := transform(points[i+1])
		r.drawLine(img, x1, y1, x2, y2, r.LineColor)
	}
}

func (r *GeometryRenderer) drawPolygon(img *image.RGBA, poly Polygon, transform func(Point) (int, int)) {
	// Draw fill for exterior ring (simple scanline fill)
	if len(poly.Rings) > 0 && len(poly.Rings[0]) > 2 {
		r.fillPolygon(img, poly.Rings[0], transform)
	}

	// Draw outline for all rings
	for _, ring := range poly.Rings {
		if len(ring) > 1 {
			r.drawLineString(img, ring, transform)
			// Close the ring
			if len(ring) > 2 {
				x1, y1 := transform(ring[len(ring)-1])
				x2, y2 := transform(ring[0])
				r.drawLine(img, x1, y1, x2, y2, r.LineColor)
			}
		}
	}
}

func (r *GeometryRenderer) fillPolygon(img *image.RGBA, points []Point, transform func(Point) (int, int)) {
	if len(points) < 3 {
		return
	}

	// Transform all points
	transformed := make([]struct{ x, y int }, len(points))
	minY, maxY := r.Height, 0
	for i, p := range points {
		transformed[i].x, transformed[i].y = transform(p)
		if transformed[i].y < minY {
			minY = transformed[i].y
		}
		if transformed[i].y > maxY {
			maxY = transformed[i].y
		}
	}

	// Scanline fill
	for y := minY; y <= maxY; y++ {
		var intersections []int
		n := len(transformed)
		for i := 0; i < n; i++ {
			j := (i + 1) % n
			y1, y2 := transformed[i].y, transformed[j].y
			x1, x2 := transformed[i].x, transformed[j].x

			if (y1 <= y && y < y2) || (y2 <= y && y < y1) {
				x := x1 + (y-y1)*(x2-x1)/(y2-y1)
				intersections = append(intersections, x)
			}
		}

		// Sort intersections
		for i := 0; i < len(intersections)-1; i++ {
			for j := i + 1; j < len(intersections); j++ {
				if intersections[i] > intersections[j] {
					intersections[i], intersections[j] = intersections[j], intersections[i]
				}
			}
		}

		// Fill between pairs
		for i := 0; i+1 < len(intersections); i += 2 {
			for x := intersections[i]; x <= intersections[i+1]; x++ {
				if x >= 0 && x < r.Width && y >= 0 && y < r.Height {
					img.Set(x, y, r.FillColor)
				}
			}
		}
	}
}

// Bresenham's line algorithm
func (r *GeometryRenderer) drawLine(img *image.RGBA, x1, y1, x2, y2 int, c color.Color) {
	dx := abs(x2 - x1)
	dy := abs(y2 - y1)
	sx, sy := 1, 1
	if x1 >= x2 {
		sx = -1
	}
	if y1 >= y2 {
		sy = -1
	}
	err := dx - dy

	for {
		if x1 >= 0 && x1 < r.Width && y1 >= 0 && y1 < r.Height {
			img.Set(x1, y1, c)
		}
		if x1 == x2 && y1 == y2 {
			break
		}
		e2 := 2 * err
		if e2 > -dy {
			err -= dy
			x1 += sx
		}
		if e2 < dx {
			err += dx
			y1 += sy
		}
	}
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// parseGeometry parses WKB (hex) or WKT geometry strings
func parseGeometry(val string) (interface{}, error) {
	val = strings.TrimSpace(val)
	if val == "" || val == "NULL" {
		return nil, nil
	}

	// Try WKB (hex) first - starts with hex digits
	if isHexString(val) {
		return parseWKB(val)
	}

	// Try WKT
	if strings.HasPrefix(strings.ToUpper(val), "POINT") ||
		strings.HasPrefix(strings.ToUpper(val), "LINESTRING") ||
		strings.HasPrefix(strings.ToUpper(val), "POLYGON") ||
		strings.HasPrefix(strings.ToUpper(val), "MULTI") ||
		strings.HasPrefix(strings.ToUpper(val), "GEOMETRY") {
		return parseWKT(val)
	}

	return nil, fmt.Errorf("unknown geometry format")
}

func isHexString(s string) bool {
	if len(s) < 2 {
		return false
	}
	for _, c := range s {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return true
}

// parseWKB parses Well-Known Binary (hex encoded)
func parseWKB(hexStr string) (interface{}, error) {
	data, err := hex.DecodeString(hexStr)
	if err != nil {
		return nil, err
	}

	if len(data) < 5 {
		return nil, fmt.Errorf("WKB too short")
	}

	// First byte is byte order (0=big endian, 1=little endian)
	var byteOrder binary.ByteOrder = binary.LittleEndian
	if data[0] == 0 {
		byteOrder = binary.BigEndian
	}

	// Next 4 bytes are geometry type
	geomType := byteOrder.Uint32(data[1:5])

	// Handle EWKB (PostGIS extended WKB with SRID)
	offset := 5
	if geomType&0x20000000 != 0 { // Has SRID
		offset += 4 // Skip SRID
		geomType = geomType & 0x0FFFFFFF
	}
	if geomType&0x80000000 != 0 { // Has Z
		geomType = geomType & 0x7FFFFFFF
	}
	if geomType&0x40000000 != 0 { // Has M
		geomType = geomType & 0xBFFFFFFF
	}

	reader := bytes.NewReader(data[offset:])

	switch geomType {
	case 1: // Point
		return readWKBPoint(reader, byteOrder)
	case 2: // LineString
		return readWKBLineString(reader, byteOrder)
	case 3: // Polygon
		return readWKBPolygon(reader, byteOrder)
	case 4: // MultiPoint
		return readWKBMultiPoint(reader, byteOrder, data[0])
	case 5: // MultiLineString
		return readWKBMultiLineString(reader, byteOrder, data[0])
	case 6: // MultiPolygon
		return readWKBMultiPolygon(reader, byteOrder, data[0])
	default:
		return nil, fmt.Errorf("unsupported WKB geometry type: %d", geomType)
	}
}

func readWKBPoint(r *bytes.Reader, order binary.ByteOrder) (Point, error) {
	var x, y float64
	if err := binary.Read(r, order, &x); err != nil {
		return Point{}, err
	}
	if err := binary.Read(r, order, &y); err != nil {
		return Point{}, err
	}
	return Point{X: x, Y: y}, nil
}

func readWKBLineString(r *bytes.Reader, order binary.ByteOrder) (LineString, error) {
	var numPoints uint32
	if err := binary.Read(r, order, &numPoints); err != nil {
		return LineString{}, err
	}

	points := make([]Point, numPoints)
	for i := uint32(0); i < numPoints; i++ {
		p, err := readWKBPoint(r, order)
		if err != nil {
			return LineString{}, err
		}
		points[i] = p
	}
	return LineString{Points: points}, nil
}

func readWKBPolygon(r *bytes.Reader, order binary.ByteOrder) (Polygon, error) {
	var numRings uint32
	if err := binary.Read(r, order, &numRings); err != nil {
		return Polygon{}, err
	}

	rings := make([][]Point, numRings)
	for i := uint32(0); i < numRings; i++ {
		var numPoints uint32
		if err := binary.Read(r, order, &numPoints); err != nil {
			return Polygon{}, err
		}
		ring := make([]Point, numPoints)
		for j := uint32(0); j < numPoints; j++ {
			p, err := readWKBPoint(r, order)
			if err != nil {
				return Polygon{}, err
			}
			ring[j] = p
		}
		rings[i] = ring
	}
	return Polygon{Rings: rings}, nil
}

func readWKBMultiPoint(r *bytes.Reader, order binary.ByteOrder, byteOrderByte byte) (MultiPoint, error) {
	var numGeoms uint32
	if err := binary.Read(r, order, &numGeoms); err != nil {
		return MultiPoint{}, err
	}

	points := make([]Point, 0, numGeoms)
	for i := uint32(0); i < numGeoms; i++ {
		// Skip byte order and type for each geometry
		r.ReadByte()
		var geomType uint32
		binary.Read(r, order, &geomType)

		p, err := readWKBPoint(r, order)
		if err != nil {
			return MultiPoint{}, err
		}
		points = append(points, p)
	}
	return MultiPoint{Points: points}, nil
}

func readWKBMultiLineString(r *bytes.Reader, order binary.ByteOrder, byteOrderByte byte) (MultiLineString, error) {
	var numGeoms uint32
	if err := binary.Read(r, order, &numGeoms); err != nil {
		return MultiLineString{}, err
	}

	lines := make([]LineString, 0, numGeoms)
	for i := uint32(0); i < numGeoms; i++ {
		// Skip byte order and type
		r.ReadByte()
		var geomType uint32
		binary.Read(r, order, &geomType)

		line, err := readWKBLineString(r, order)
		if err != nil {
			return MultiLineString{}, err
		}
		lines = append(lines, line)
	}
	return MultiLineString{Lines: lines}, nil
}

func readWKBMultiPolygon(r *bytes.Reader, order binary.ByteOrder, byteOrderByte byte) (MultiPolygon, error) {
	var numGeoms uint32
	if err := binary.Read(r, order, &numGeoms); err != nil {
		return MultiPolygon{}, err
	}

	polygons := make([]Polygon, 0, numGeoms)
	for i := uint32(0); i < numGeoms; i++ {
		// Skip byte order and type
		r.ReadByte()
		var geomType uint32
		binary.Read(r, order, &geomType)

		poly, err := readWKBPolygon(r, order)
		if err != nil {
			return MultiPolygon{}, err
		}
		polygons = append(polygons, poly)
	}
	return MultiPolygon{Polygons: polygons}, nil
}

// parseWKT parses Well-Known Text
func parseWKT(wkt string) (interface{}, error) {
	wkt = strings.TrimSpace(wkt)
	upper := strings.ToUpper(wkt)

	if strings.HasPrefix(upper, "POINT") {
		return parseWKTPoint(wkt)
	} else if strings.HasPrefix(upper, "LINESTRING") {
		return parseWKTLineString(wkt)
	} else if strings.HasPrefix(upper, "POLYGON") {
		return parseWKTPolygon(wkt)
	} else if strings.HasPrefix(upper, "MULTIPOINT") {
		return parseWKTMultiPoint(wkt)
	} else if strings.HasPrefix(upper, "MULTILINESTRING") {
		return parseWKTMultiLineString(wkt)
	} else if strings.HasPrefix(upper, "MULTIPOLYGON") {
		return parseWKTMultiPolygon(wkt)
	}

	return nil, fmt.Errorf("unsupported WKT geometry type")
}

func parseWKTPoint(wkt string) (Point, error) {
	// POINT(x y) or POINT (x y)
	start := strings.Index(wkt, "(")
	end := strings.LastIndex(wkt, ")")
	if start == -1 || end == -1 {
		return Point{}, fmt.Errorf("invalid POINT WKT")
	}

	coords := strings.Fields(strings.TrimSpace(wkt[start+1 : end]))
	if len(coords) < 2 {
		return Point{}, fmt.Errorf("invalid POINT coordinates")
	}

	var x, y float64
	fmt.Sscanf(coords[0], "%f", &x)
	fmt.Sscanf(coords[1], "%f", &y)
	return Point{X: x, Y: y}, nil
}

func parseWKTLineString(wkt string) (LineString, error) {
	start := strings.Index(wkt, "(")
	end := strings.LastIndex(wkt, ")")
	if start == -1 || end == -1 {
		return LineString{}, fmt.Errorf("invalid LINESTRING WKT")
	}

	points, err := parseCoordinateList(wkt[start+1 : end])
	if err != nil {
		return LineString{}, err
	}
	return LineString{Points: points}, nil
}

func parseWKTPolygon(wkt string) (Polygon, error) {
	start := strings.Index(wkt, "((")
	end := strings.LastIndex(wkt, "))")
	if start == -1 || end == -1 {
		return Polygon{}, fmt.Errorf("invalid POLYGON WKT")
	}

	content := wkt[start+2 : end]
	ringStrs := strings.Split(content, "),(")

	rings := make([][]Point, len(ringStrs))
	for i, ringStr := range ringStrs {
		ringStr = strings.Trim(ringStr, "()")
		points, err := parseCoordinateList(ringStr)
		if err != nil {
			return Polygon{}, err
		}
		rings[i] = points
	}
	return Polygon{Rings: rings}, nil
}

func parseWKTMultiPoint(wkt string) (MultiPoint, error) {
	start := strings.Index(wkt, "(")
	end := strings.LastIndex(wkt, ")")
	if start == -1 || end == -1 {
		return MultiPoint{}, fmt.Errorf("invalid MULTIPOINT WKT")
	}

	content := wkt[start+1 : end]
	// Handle both MULTIPOINT(x y, x y) and MULTIPOINT((x y), (x y)) formats
	content = strings.ReplaceAll(content, "(", "")
	content = strings.ReplaceAll(content, ")", "")

	points, err := parseCoordinateList(content)
	if err != nil {
		return MultiPoint{}, err
	}
	return MultiPoint{Points: points}, nil
}

func parseWKTMultiLineString(wkt string) (MultiLineString, error) {
	start := strings.Index(wkt, "((")
	end := strings.LastIndex(wkt, "))")
	if start == -1 || end == -1 {
		return MultiLineString{}, fmt.Errorf("invalid MULTILINESTRING WKT")
	}

	content := wkt[start+2 : end]
	lineStrs := strings.Split(content, "),(")

	lines := make([]LineString, len(lineStrs))
	for i, lineStr := range lineStrs {
		lineStr = strings.Trim(lineStr, "()")
		points, err := parseCoordinateList(lineStr)
		if err != nil {
			return MultiLineString{}, err
		}
		lines[i] = LineString{Points: points}
	}
	return MultiLineString{Lines: lines}, nil
}

func parseWKTMultiPolygon(wkt string) (MultiPolygon, error) {
	start := strings.Index(wkt, "(((")
	end := strings.LastIndex(wkt, ")))")
	if start == -1 || end == -1 {
		return MultiPolygon{}, fmt.Errorf("invalid MULTIPOLYGON WKT")
	}

	content := wkt[start+3 : end]
	polyStrs := strings.Split(content, ")),((")

	polygons := make([]Polygon, len(polyStrs))
	for i, polyStr := range polyStrs {
		polyStr = strings.Trim(polyStr, "()")
		ringStrs := strings.Split(polyStr, "),(")

		rings := make([][]Point, len(ringStrs))
		for j, ringStr := range ringStrs {
			ringStr = strings.Trim(ringStr, "()")
			points, err := parseCoordinateList(ringStr)
			if err != nil {
				return MultiPolygon{}, err
			}
			rings[j] = points
		}
		polygons[i] = Polygon{Rings: rings}
	}
	return MultiPolygon{Polygons: polygons}, nil
}

func parseCoordinateList(s string) ([]Point, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}

	coordPairs := strings.Split(s, ",")
	points := make([]Point, 0, len(coordPairs))

	for _, pair := range coordPairs {
		pair = strings.TrimSpace(pair)
		coords := strings.Fields(pair)
		if len(coords) >= 2 {
			var x, y float64
			fmt.Sscanf(coords[0], "%f", &x)
			fmt.Sscanf(coords[1], "%f", &y)
			points = append(points, Point{X: x, Y: y})
		}
	}
	return points, nil
}

// RenderGeometriesToPNG renders geometries and returns base64-encoded PNG data
func RenderGeometriesToPNG(geomValues []string, width, height int) (string, error) {
	renderer := NewGeometryRenderer(width, height)
	pngData, err := renderer.RenderGeometries(geomValues)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(pngData), nil
}

// Base64ToKittyGraphics converts base64 PNG data to Kitty graphics protocol escape sequence
func Base64ToKittyGraphics(b64Data string) string {
	if b64Data == "" {
		return ""
	}

	var result strings.Builder

	// Build Kitty graphics protocol escape sequence
	// Format: \033_Ga=T,f=100,... ;base64data\033\\
	// a=T means transmit and display
	// f=100 means PNG format
	// We'll send in chunks if needed

	chunkSize := 4096
	for i := 0; i < len(b64Data); i += chunkSize {
		end := i + chunkSize
		if end > len(b64Data) {
			end = len(b64Data)
		}
		chunk := b64Data[i:end]

		isFirst := i == 0
		isLast := end >= len(b64Data)

		if isFirst && isLast {
			// Single chunk
			result.WriteString(fmt.Sprintf("\033_Ga=T,f=100;%s\033\\", chunk))
		} else if isFirst {
			// First chunk of multi
			result.WriteString(fmt.Sprintf("\033_Ga=T,f=100,m=1;%s\033\\", chunk))
		} else if isLast {
			// Last chunk
			result.WriteString(fmt.Sprintf("\033_Gm=0;%s\033\\", chunk))
		} else {
			// Middle chunk
			result.WriteString(fmt.Sprintf("\033_Gm=1;%s\033\\", chunk))
		}
	}

	return result.String()
}

// RenderToKittyGraphics renders geometries and returns Kitty graphics protocol escape sequence
func RenderToKittyGraphics(geomValues []string, width, height int) (string, error) {
	b64Data, err := RenderGeometriesToPNG(geomValues, width, height)
	if err != nil {
		return "", err
	}
	return Base64ToKittyGraphics(b64Data), nil
}

// DetectGeometryColumn finds geometry column index in results
func DetectGeometryColumn(columns []string, sampleRow []string) int {
	// First check column names for common geometry column names
	geomNames := []string{"geom", "geometry", "the_geom", "wkb_geometry", "shape", "geo"}
	for i, col := range columns {
		colLower := strings.ToLower(col)
		for _, name := range geomNames {
			if colLower == name || strings.Contains(colLower, "geom") {
				return i
			}
		}
	}

	// Check sample data for geometry-looking values
	if len(sampleRow) > 0 {
		for i, val := range sampleRow {
			if isGeometryValue(val) {
				return i
			}
		}
	}

	return -1
}

// isGeometryValue checks if a value looks like geometry data
func isGeometryValue(val string) bool {
	val = strings.TrimSpace(val)
	if val == "" || val == "NULL" {
		return false
	}

	// Check for WKB (hex) - long hex string starting with 01 (little endian) or 00 (big endian)
	if len(val) > 20 && isHexString(val) {
		return true
	}

	// Check for WKT
	upper := strings.ToUpper(val)
	return strings.HasPrefix(upper, "POINT") ||
		strings.HasPrefix(upper, "LINESTRING") ||
		strings.HasPrefix(upper, "POLYGON") ||
		strings.HasPrefix(upper, "MULTI") ||
		strings.HasPrefix(upper, "GEOMETRY")
}
