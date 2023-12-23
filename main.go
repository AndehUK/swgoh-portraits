package main

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"

	"golang.org/x/image/font"

	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

type CharacterPortrait struct {
	Character  string
	GearLevel  int
	RelicLevel int
	Zetas      int
	Omicrons   int
}

type Character struct {
	Name        string
	Affiliation string
	imgSrc      string
	maxZetas    int
	maxOmicrons int
}

// List of supported characters
var supportedCharacters = map[string]Character{
	"darth_vader": {
		Name:        "Darth Vader",
		Affiliation: "dark_side",
		imgSrc:      "darth_vader.png",
		maxZetas:    3,
		maxOmicrons: 1,
	},
}

// Load the Inter font
func loadFont(path string, size float64) (font.Face, error) {
	fontBytes, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	fontType, err := opentype.Parse(fontBytes)
	if err != nil {
		return nil, err
	}
	face, err := opentype.NewFace(fontType, &opentype.FaceOptions{
		Size:    size,
		DPI:     72,
		Hinting: font.HintingFull,
	})
	if err != nil {
		return nil, err
	}
	return face, nil
}

// Draw text onto an image at a specified position
func drawText(img *image.RGBA, face font.Face, x, y int, text string, col color.Color) {
	point := fixed.P(x, y)
	d := &font.Drawer{
		Dst:  img,
		Src:  image.NewUniform(col),
		Face: face,
		Dot:  point,
	}
	d.DrawString(text)
}

func main() {
	http.HandleFunc("/create", createPortraitHandler)

	fmt.Println("Server is running on http://localhost:3000")
	if err := http.ListenAndServe(":3000", nil); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

func createPortraitHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Only GET method is allowed", http.StatusMethodNotAllowed)
		return
	}

	query := r.URL.Query()
	charID := query.Get("char")

	char, ok := supportedCharacters[charID]
	if !ok {
		http.Error(w, fmt.Sprintf("Character '%s' is not supported by this API", charID), http.StatusBadRequest)
		return
	}

	gearLevel, err := getIntFromQuery(query, "gear_level")
	if err != nil || gearLevel < 1 || gearLevel > 13 {
		http.Error(w, "The gear_level must be between 1 and 13", http.StatusBadRequest)
		return
	}

	relicLevel, err := getIntFromQuery(query, "relic_level")
	if gearLevel != 13 && relicLevel != 0 && err == nil {
		http.Error(w, "The relic_level should not be provided if gear_level is not 13", http.StatusBadRequest)
		return
	}
	if gearLevel == 13 && (relicLevel < 1 || relicLevel > 9) {
		http.Error(w, "The relic_level must be between 1 and 9", http.StatusBadRequest)
		return
	}

	zetas, _ := getIntFromQuery(query, "zetas") // Error ignored to allow default value of 0
	if zetas < 0 || zetas > char.maxZetas {
		http.Error(w, fmt.Sprintf("The zeta level must be between 0 and %d for %s", char.maxZetas, char.Name), http.StatusBadRequest)
		return
	}

	omicrons, _ := getIntFromQuery(query, "omicrons") // Error ignored to allow default value of 0
	if omicrons < 0 || omicrons > char.maxOmicrons {
		http.Error(w, fmt.Sprintf("The omicron level must be between 0 and %d for %s", char.maxOmicrons, char.Name), http.StatusBadRequest)
		return
	}

	portrait := CharacterPortrait{
		Character:  charID,
		GearLevel:  gearLevel,
		RelicLevel: relicLevel,
		Zetas:      zetas,
		Omicrons:   omicrons,
	}

	img, err := buildPortrait(portrait, char)
	if err != nil {
		http.Error(w, "Failed to create portrait: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "image/png")
	if err := png.Encode(w, img); err != nil {
		http.Error(w, "Failed to encode image: "+err.Error(), http.StatusInternalServerError)
		return
	}
}

// getIntFromQuery is a helper function to get integer values from query parameters.
// It returns the integer value and an error if the conversion fails or the parameter is not present.
func getIntFromQuery(query map[string][]string, key string) (int, error) {
	valueStr, ok := query[key]
	if !ok || len(valueStr[0]) == 0 {
		// If the parameter is missing or empty, return 0 instead of an error
		return 0, nil
	}
	value, err := strconv.Atoi(valueStr[0])
	if err != nil {
		return 0, fmt.Errorf("parameter '%s' should be an integer, got '%s'", key, valueStr[0])
	}
	return value, nil
}

func buildPortrait(portrait CharacterPortrait, charData Character) (image.Image, error) {
	interFontFaceSmall, err := loadFont("assets/fonts/Inter-Regular.ttf", 18)
	if err != nil {
		return nil, err
	}

	interFontFaceLarge, err := loadFont("assets/fonts/Inter-Regular.ttf", 24)
	if err != nil {
		return nil, err
	}

	zetaBadgePosition := image.Point{18, 100}
	zetaBadgeSize := image.Point{60, 60}
	omicronBadgePosition := image.Point{121, 100}
	omicronBadgeSize := image.Point{60, 60}
	levelBadgePosition := image.Point{75, 128}
	levelBadgeSize := image.Point{50, 44}

	// Calculate text drawing positions
	zetaText := strconv.Itoa(portrait.Zetas)
	zetaTextWidth := font.MeasureString(interFontFaceSmall, zetaText).Round()
	zetaTextPosition := image.Point{
		X: zetaBadgePosition.X + (zetaBadgeSize.X-zetaTextWidth)/2,
		Y: (zetaBadgePosition.Y + (zetaBadgeSize.Y+interFontFaceSmall.Metrics().Ascent.Ceil())/2) - 4,
	}

	omicronText := strconv.Itoa(portrait.Omicrons)
	omicronTextWidth := font.MeasureString(interFontFaceSmall, omicronText).Round()
	omicronTextPosition := image.Point{
		X: omicronBadgePosition.X + (omicronBadgeSize.X-omicronTextWidth)/2,
		Y: (omicronBadgePosition.Y + (omicronBadgeSize.Y+interFontFaceSmall.Metrics().Ascent.Ceil())/2) - 4,
	}

	levelText := "85" // The level is hardcoded in this example; you may want to make this dynamic
	levelTextWidth := font.MeasureString(interFontFaceLarge, levelText).Round()
	levelTextPosition := image.Point{
		X: levelBadgePosition.X + (levelBadgeSize.X-levelTextWidth)/2,
		Y: (levelBadgePosition.Y + (levelBadgeSize.Y+interFontFaceLarge.Metrics().Ascent.Ceil())/2) - 5,
	}

	// Load and center the character image on a 200x200 canvas
	characterImg, err := loadImage("assets/characters/" + charData.imgSrc)
	if err != nil {
		return nil, err
	}
	finalImage, err := placeImageOnCanvas(characterImg)
	if err != nil {
		return nil, err
	}

	// Add gear or relic border based on GearLevel
	if portrait.GearLevel < 13 {
		borderImg, err := loadImage("assets/gear/" + strconv.Itoa(portrait.GearLevel) + ".png")
		if err != nil {
			return nil, err
		}
		centerImageOnCanvas(borderImg, finalImage)
	} else {
		// Load and draw relic border for gear level 13
		// Assuming relic border path is similar to gear
		relicBorderImg, err := loadImage("assets/relics/" + charData.Affiliation + ".png") // Placeholder path
		if err != nil {
			return nil, err
		}
		draw.Draw(finalImage, finalImage.Bounds(), relicBorderImg, image.Point{}, draw.Over)
	}

	// Add character level badge and level number text
	levelBadgeImg, err := loadImage("assets/badges/level.png")
	if err != nil {
		return nil, err
	}
	draw.Draw(finalImage, levelBadgeImg.Bounds().Add(levelBadgePosition), levelBadgeImg, image.Point{}, draw.Over)
	drawText(finalImage, interFontFaceLarge, levelTextPosition.X, levelTextPosition.Y, levelText, color.White)

	// Conditionally add zeta badge
	if portrait.Zetas > 0 {
		zetaBadgeImg, err := loadImage("assets/badges/zeta.png")
		if err != nil {
			return nil, err
		}
		draw.Draw(finalImage, zetaBadgeImg.Bounds().Add(zetaBadgePosition), zetaBadgeImg, image.Point{}, draw.Over)
		drawText(finalImage, interFontFaceSmall, zetaTextPosition.X, zetaTextPosition.Y, zetaText, color.White)
	}

	// Conditionally add omicron badge
	if portrait.Omicrons > 0 {
		omicronBadgeImg, err := loadImage("assets/badges/omicron.png")
		if err != nil {
			return nil, err
		}
		draw.Draw(finalImage, omicronBadgeImg.Bounds().Add(omicronBadgePosition), omicronBadgeImg, image.Point{}, draw.Over)
		drawText(finalImage, interFontFaceSmall, omicronTextPosition.X, omicronTextPosition.Y, omicronText, color.White)
	}

	// Return the composed final image
	return finalImage, nil
}

// loadImage reads an image from the file with the given path and decodes it into an image.Image
func loadImage(filePath string) (image.Image, error) {
	// Open the file
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close() // Make sure to close the file when done

	// Decode the image
	img, _, err := image.Decode(file)
	if err != nil {
		return nil, err
	}

	return img, nil
}

// placeImageOnCanvas centers an image onto a 200x200 canvas
func placeImageOnCanvas(src image.Image) (*image.RGBA, error) {
	// Create a new blank 200x200 canvas
	canvasSize := image.Point{200, 200}
	canvasRect := image.Rectangle{image.Point{0, 0}, canvasSize}
	canvas := image.NewRGBA(canvasRect)

	// Initialize canvas to be transparent or a background color
	clearColor := image.Transparent // Use image.Transparent for a transparent background
	draw.Draw(canvas, canvas.Bounds(), &image.Uniform{clearColor}, image.Point{}, draw.Src)

	// Calculate the position to center src on the canvas
	srcBounds := src.Bounds()
	srcSize := srcBounds.Size()
	offset := canvasSize.Sub(srcSize).Div(2) // Centering formula

	// The point on the canvas where the src image will be drawn
	drawPoint := image.Point{
		X: offset.X,
		Y: offset.Y,
	}

	// Draw the src image onto the canvas, centered
	draw.Draw(canvas, srcBounds.Add(drawPoint), src, srcBounds.Min, draw.Over)

	return canvas, nil
}

func centerImageOnCanvas(src image.Image, dest *image.RGBA) {
	// Calculate the position to center src on dest
	srcBounds := src.Bounds()
	destBounds := dest.Bounds()
	srcSize := srcBounds.Size()
	destSize := destBounds.Size()
	offset := image.Pt((destSize.X-srcSize.X)/2, (destSize.Y-srcSize.Y)/2)

	// The point on the canvas where the src image will be drawn
	drawPoint := image.Point{
		X: offset.X,
		Y: offset.Y,
	}

	// Draw the src image onto the dest, centered
	draw.Draw(dest, srcBounds.Add(drawPoint), src, srcBounds.Min, draw.Over)
}
