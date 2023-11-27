package main

import (
	_ "embed"
	"encoding/base64"
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
    --size=<size>  Diagram size (height and width) in pixels (default: 400)
    --bg=<color>   Background color as hexadecimal RRGGBB (default: FFFFFF)
    --fg=<color>   Foreground color as hexadecimal RRGGBB (default: 000000)
    --grayscale    Output grayscale PNG
    --base64       Base64 output
    --coordinates  Show coordinates on the diagram
    --flip         Flip the diagram
    --auto-flip    Flip the diagram if Black to move
Positional arguments:
    <fen>          FEN record
    <output-file>  Output file name or "-" for the stdout
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
	numbers: [8]rune{
		'\uf0c7', '\uf0c6', '\uf0c5', '\uf0c4', // 8, 7, 6, 5
		'\uf0c3', '\uf0c2', '\uf0c1', '\uf0c0', // 4, 3, 2, 1
	},
	letters: [8]rune{
		'\uf0c8', '\uf0c9', '\uf0ca', '\uf0cb', // a, b, c, d
		'\uf0cc', '\uf0cd', '\uf0ce', '\uf0cf', // e, f, g, h
	},
	topLeftCorner:     '\uf031',
	topSide:           '\uf032',
	topRightCorner:    '\uf033',
	leftSide:          '\uf034',
	rightSide:         '\uf035',
	bottomLeftCorner:  '\uf037',
	bottomSide:        '\uf038',
	bottomRightCorner: '\uf039',
}

func check(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "fen2png: error: %v\n", err)
		os.Exit(1)
	}
}

func reverse(s string) string {
	runes := []rune(s)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	return string(runes)
}

func decodeFEN(fen string, f *chessFont, opts *options) (rows []string, err error) {
	fields := strings.Fields(fen)
	if len(fields) == 0 {
		return nil, fmt.Errorf("empty FEN")
	}

	if opts.autoFlip && len(fields) < 2 {
		return nil, fmt.Errorf("the second field of FEN is required for auto-flip")
	}
	flip := opts.flip || (opts.autoFlip && fields[1] == "b")
	if flip {
		fields[0] = reverse(fields[0])
	}

	ranks := strings.Split(fields[0], "/")
	if len(ranks) != 8 {
		return nil, fmt.Errorf("%d ranks in FEN", len(ranks))
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
			if flip {
				row.WriteRune(f.numbers[7-y])
			} else {
				row.WriteRune(f.numbers[y])
			}
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
					return nil, fmt.Errorf("unknown piece %q in FEN", piece)
				}
			}
		}
		if x != 8 {
			return nil, fmt.Errorf("%d files in FEN at rank %q", x, rank)
		}
		row.WriteRune(f.rightSide)
		rows = append(rows, row.String())
	}

	// Bottom
	row.Reset()
	row.WriteRune(f.bottomLeftCorner)
	for i := 0; i < 8; i++ {
		if opts.coordinates {
			if flip {
				row.WriteRune(f.letters[7-i])
			} else {
				row.WriteRune(f.letters[i])
			}
		} else {
			row.WriteRune(f.bottomSide)
		}
	}
	row.WriteRune(f.bottomRightCorner)
	rows = append(rows, row.String())

	return rows, nil
}

type options struct {
	size        int
	bg, fg      color.Color
	grayscale   bool
	base64      bool
	coordinates bool
	flip        bool
	autoFlip    bool
	fen         string
	outputFile  string
	help        bool
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
		case "--coordinates":
			opts.coordinates = true
		case "--flip":
			opts.flip = true
		case "--auto-flip":
			opts.autoFlip = true
		case "--help":
			opts.help = true
			return opts, nil
		default:
			return nil, fmt.Errorf("unrecognized option: %q", option)
		}
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

	rows, err := decodeFEN(opts.fen, &merida, opts)
	check(err)
	height := fixed.Int26_6(fontSize * 64)
	currentHeight := height
	for _, row := range rows {
		_, err = ctx.DrawString(row, fixed.Point26_6{0, currentHeight})
		check(err)
		currentHeight += height
	}

	var output io.WriteCloser
	if opts.outputFile == "-" {
		output = os.Stdout
	} else {
		output, err = os.Create(opts.outputFile)
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
