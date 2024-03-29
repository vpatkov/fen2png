A simple, single-binary program that creates a PNG image of a chess diagram
from its FEN record. The diagram style follows the one commonly used in printed
books.

![](example.png "r3qb1k/1b4p1/p2pr2p/3n4/Pnp1N1N1/6RP/1B3PP1/1B1QR1K1 w")

For drawing, the program uses a TTF chess font (Merida), so creates
high-quality output images independent of size.

FEN notation is extended with dots `d` and crosses `x` to mark squares.

The program is created with the goal of rendering FEN in Markdown documents.
A Lua filter for Pandoc is included: it replaces code blocks that have `fen`
syntax with the diagrams, embedded in base64 (as data URL). A FEN record should
be written on the first line of the code block, options for the fen2png can be
written on the second line.


## Usage

```
Usage: fen2png [options] <fen> <output-file>
Options:
    --size=<size>  Diagram size (height and width) in pixels (default: 400)
    --bg=<color>   Background color (default: FFFFFF)
    --fg=<color>   Foreground color (default: 0)
    --base64       Base64 output
    --coordinates  Show coordinates on the diagram
    --flip         Flip the diagram
    --auto-flip    Flip the diagram if Black to move

Color must be a hexadecimal number with the four octets from the most to the
least significant represents transparency, red, green, and blue components
respectively. Transparency of 0 means fully opaque.

Positional arguments:
    <fen>          FEN record
    <output-file>  Output file name or "-" for the stdout
```


## License

The program uses a freeware chess font "Merida" created by Armando Hernandez
Marroquin. The program itself dedicated to the public domain.

[![CC0 button](https://licensebuttons.net/p/zero/1.0/88x31.png)](http://creativecommons.org/publicdomain/zero/1.0/)
