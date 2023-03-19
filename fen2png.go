package main

import (
	_ "embed"
	"encoding/base64"
	"encoding/csv"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/golang/freetype"
	"github.com/golang/freetype/truetype"
	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
)

const helpMessage = `Usage: fen2png [options] <fen> <output-file>
Options:
    --size=<size>     Diagram size (height and width) in pixels (default: 400)
    --bg=<color>      Background color as hexadecimal RRGGBB (default: FFFFFF)
    --fg=<color>      Foreground color as hexadecimal RRGGBB (default: 000000)
    --grayscale       Output grayscale PNG
    --base64          Base64 output
    --from-file       Parse CSV file <fen>,<output-file>
    --turn-indicator  Show black dot in top right when it's black's turn
    --coordinates     Add letters and numbers to chessboard
Positional arguments:
    <fen>             FEN record (only the first field is mandatory)
    <output-file>     Output file name or "-" for the stdout
`

type chessFont struct {
	ttf     []byte
	pieces  map[rune][2]rune
	numbers [8]rune
	letters [8]rune
	topLeftCorner,
	topSide,
	topRightCorner,
	leftSide,
	rightSide,
	bottomLeftCorner,
	bottomSide,
	turnIndicator,
	bottomRightCorner rune
}

//go:embed merida.ttf
var meridaTTF []byte

var merida = chessFont{
	ttf: meridaTTF,
	pieces: map[rune][2]rune{
		' ': {'\uf020', '\uf02b'}, // No piece on light and dark squares
		'R': {'\uf072', '\uf052'}, // White rook on light and dark squares
		'N': {'\uf06e', '\uf04e'}, // White knight on light and dark squares
		'B': {'\uf062', '\uf042'}, // White bishop on light and dark squares
		'Q': {'\uf071', '\uf051'}, // White queen on light and dark squares
		'K': {'\uf06b', '\uf04b'}, // White king on light and dark squares
		'P': {'\uf070', '\uf050'}, // White pawn on light and dark squares
		'r': {'\uf074', '\uf054'}, // Black rook on light and dark squares
		'n': {'\uf06d', '\uf04d'}, // Black knight on light and dark squares
		'b': {'\uf076', '\uf056'}, // Black bishop on light and dark squares
		'q': {'\uf077', '\uf057'}, // Black queen on light and dark squares
		'k': {'\uf06c', '\uf04c'}, // Black king on light and dark squares
		'p': {'\uf06f', '\uf04f'}, // Black pawn on light and dark squares
		'd': {'\uf02e', '\uf03a'}, // Black dot on light and dark squares
		'x': {'\uf078', '\uf058'}, // Black cross on light and dark squares
	},
	numbers:           [8]rune{'\uf0c7', '\uf0c6', '\uf0c5', '\uf0c4', '\uf0c3', '\uf0c2', '\uf0c1', '\uf0c0'},
	letters:           [8]rune{'\uf0c8', '\uf0c9', '\uf0ca', '\uf0cb', '\uf0cc', '\uf0cd', '\uf0ce', '\uf0cf'},
	topLeftCorner:     '\uf031',
	topSide:           '\uf032',
	topRightCorner:    '\uf033',
	leftSide:          '\uf034',
	rightSide:         '\uf035',
	bottomLeftCorner:  '\uf037',
	bottomSide:        '\uf038',
	bottomRightCorner: '\uf039',
	turnIndicator:     '\uf02e',
}

func check(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "fen2png: error: %v\n", err)
		os.Exit(1)
	}
}

func decodeFEN(fen string, f *chessFont, opts *options) (rows []string, turn string, err error) {
	fields := strings.Fields(fen)
	if len(fields) == 0 {
		return nil, "", fmt.Errorf("empty FEN")
	}
	ranks := strings.Split(fields[0], "/")
	if len(ranks) != 8 {
		return nil, "", fmt.Errorf("%d ranks in FEN", len(ranks))
	}
	turn = "white"
	if strings.Contains(fen, " b ") {
		turn = "black"
	}

	// Top
	var row strings.Builder
	row.WriteRune(f.topLeftCorner)
	for i := 0; i < 8; i++ {
		row.WriteRune(f.topSide)
	}
	row.WriteRune(f.topRightCorner)
	rows = append(rows, row.String())

	// Middle
	for y, rank := range ranks {
		row.Reset()
		if opts.coordinates {
			row.WriteRune(f.numbers[y])
		} else {
			row.WriteRune(f.leftSide)
		}
		x := 0
		for _, piece := range rank {
			if piece >= '1' && piece <= '8' {
				r := f.pieces[' ']
				for i := 0; i < int(piece)-int('0'); i++ {
					row.WriteRune(r[(x+y)%2])
					x++
				}
			} else {
				r, ok := f.pieces[piece]
				if ok {
					row.WriteRune(r[(x+y)%2])
					x++
				} else {
					return nil, "", fmt.Errorf("unknown piece %q in FEN", piece)
				}
			}
		}
		if x != 8 {
			return nil, "", fmt.Errorf("%d files in FEN at rank %q", x, rank)
		}
		row.WriteRune(f.rightSide)
		rows = append(rows, row.String())
	}

	// Bottom
	row.Reset()
	row.WriteRune(f.bottomLeftCorner)
	for i := 0; i < 8; i++ {
		if opts.coordinates {
			row.WriteRune(f.letters[i])
		} else {
			row.WriteRune(f.bottomSide)
		}
	}
	row.WriteRune(f.bottomRightCorner)
	row.WriteRune(f.turnIndicator)
	rows = append(rows, row.String())

	return rows, turn, nil
}

type options struct {
	size          int
	bg, fg        color.Color
	grayscale     bool
	base64        bool
	fen           string
	fromFile      string
	turnIndicator bool
	coordinates   bool
	outputFile    string
	help          bool
}

func parseCmdLine(args []string) (opts *options, err error) {
	opts = &options{
		size: 400,
		bg:   color.White,
		fg:   color.Black,
	}

	if len(args) == 0 {
		opts.help = true
		return opts, nil
	}

	for ; len(args) > 0 && strings.HasPrefix(args[0], "--"); args = args[1:] {
		option, value, hasValue := strings.Cut(args[0], "=")
		switch option {
		case "--size":
			if !hasValue {
				return nil, fmt.Errorf("missing value for option %q", option)
			}
			opts.size, err = strconv.Atoi(value)
			if err != nil {
				return nil, fmt.Errorf("invalid value for option %q", option)
			}
		case "--bg", "--fg":
			if !hasValue {
				return nil, fmt.Errorf("missing value for option %q", option)
			}
			hex, err := strconv.ParseUint(value, 16, 32)
			if err != nil {
				return nil, fmt.Errorf("invalid value for option %q", option)
			}
			c := color.RGBA{
				uint8((hex >> 16) & 0xff),
				uint8((hex >> 8) & 0xff),
				uint8(hex & 0xff),
				0xff,
			}
			if option == "--bg" {
				opts.bg = c
			} else {
				opts.fg = c
			}
		case "--grayscale":
			opts.grayscale = true
		case "--base64":
			opts.base64 = true
		case "--from-file":
			opts.fromFile = value
		case "--turn-indicator":
			opts.turnIndicator = true
		case "--coordinates":
			opts.coordinates = true
		case "--help":
			opts.help = true
			return opts, nil
		default:
			return nil, fmt.Errorf("unrecognized option: %q", option)
		}
	}

	if len(opts.fromFile) > 0 {
		return opts, nil
	}

	if len(args) < 1 {
		return nil, fmt.Errorf("<fen> is required")
	} else if len(args) < 2 {
		return nil, fmt.Errorf("<output-file> is required")
	}
	opts.fen = args[0]
	opts.outputFile = args[1]
	return opts, nil
}

func main() {
	opts, err := parseCmdLine(os.Args[1:])
	check(err)
	if opts.help {
		fmt.Print(helpMessage)
		os.Exit(0)
	}

	if opts.fromFile == "" {
		process(opts, opts.fen, opts.outputFile)
		return
	}

	records, err := readCsvFile(opts.fromFile)
	check(err)

	for _, record := range records {
		process(opts, record[0], record[1])
	}
}

func readCsvFile(filePath string) (records [][]string, err error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("Unable to read input file " + filePath)
	}
	defer f.Close()

	csvReader := csv.NewReader(f)
	records, err = csvReader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("Unable to parse file as CSV for " + filePath)
	}

	return records, nil
}

func process(opts *options, fen string, outputFile string) {
	var diagram draw.Image
	if opts.grayscale {
		diagram = image.NewGray(image.Rect(0, 0, opts.size, opts.size))
	} else {
		diagram = image.NewNRGBA(image.Rect(0, 0, opts.size, opts.size))
	}
	draw.Draw(diagram, diagram.Bounds(), image.NewUniform(opts.bg), image.Point{}, draw.Src)

	ctx := freetype.NewContext()
	f, err := truetype.Parse(merida.ttf)
	check(err)
	ctx.SetFont(f)
	fontSize := float64(opts.size) / 10.0
	ctx.SetFontSize(fontSize)
	ctx.SetHinting(font.HintingNone)
	ctx.SetSrc(image.NewUniform(opts.fg))
	ctx.SetDst(diagram)
	ctx.SetClip(diagram.Bounds())

	rows, turn, err := decodeFEN(fen, &merida, opts)
	check(err)
	height := fixed.Int26_6(fontSize * 64)
	currentHeight := height
	for i, row := range rows {
		_, err = ctx.DrawString(row, fixed.Point26_6{0, currentHeight})
		check(err)
		currentHeight += height
		if opts.turnIndicator && i == 0 && turn == "black" {
			_, err = ctx.DrawString(string(merida.turnIndicator), fixed.Point26_6{(height * 9) - fixed.Int26_6(opts.size/2), currentHeight - (height / 3)})
			check(err)
		}
	}

	var output io.WriteCloser
	if outputFile == "-" {
		output = os.Stdout
	} else {
		output, err = os.Create(outputFile)
		check(err)
	}

	if opts.base64 {
		output = base64.NewEncoder(base64.StdEncoding, output)
	}

	err = png.Encode(output, diagram)
	check(err)
	err = output.Close()
	check(err)
}
