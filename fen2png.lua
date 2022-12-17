function CodeBlock(block)
    if block.classes[1] == "fen" then
        local fen = block.text:match("[^\n]+")
        local png = pandoc.pipe("fen2png", {"--base64", "--grayscale", fen, "-"}, "")
        return pandoc.Para {
            pandoc.Image("", "data:image/png;base64," .. png, fen)
        }
    end
end
