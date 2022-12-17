TARGET = fen2png
PREFIX = /usr/local

all: $(TARGET)

$(TARGET): $(TARGET).go
	go build -o $@ $<

tidy:
	go mod tidy
	go fmt

clean:
	rm -f $(TARGET) *~

install:
	install -m 755 -t $(PREFIX)/bin $(TARGET)

uninstall:
	rm -f $(PREFIX)/bin/$(TARGET)

.PHONY: all tidy clean install uninstall
