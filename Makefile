.PHONY: install

# Build and install to /opt/homebrew/bin (avoids macOS AMFI code-signing cache)
install:
	go build -ldflags="-s -w" -o /tmp/diffpane .
	rm -f /opt/homebrew/bin/diffpane
	mv /tmp/diffpane /opt/homebrew/bin/diffpane
