function CodeBlock(block)
    if block.classes[1] == "fen" then
        local fen = block.text:match("[^\n]+")
        local opts = block.text:match("\n([^\n]+)")

        local args = {}
        if opts ~= nil then
            for opt in opts:gmatch("%-%-%S+") do
                table.insert(args, opt)
            end
        end

        if next(args) == nil then
            table.insert(args, "--grayscale")
        end

        table.insert(args, "--base64")
        table.insert(args, fen)
        table.insert(args, "-")

        local ok, png = pcall(pandoc.pipe, "fen2png", args, "")
        if ok then
            return pandoc.Para {
                pandoc.Image("", "data:image/png;base64," .. png, fen)
            }
        end
    end
end
