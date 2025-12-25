# sucker
File sucker – sucks the files. Files. Files from the Internet.

## Usage

### Basic Usage
```bash
# Download files from HTML page
curl https://site.com/pages/1.html | ./sucker

# Or from clipboard (macOS)
pbpaste | ./sucker
```

### Adding Downloads While Running
While `sucker` is downloading files, you can dynamically add new downloads from your clipboard:

1. Start `sucker` normally (it will begin downloading)
2. Copy new links or HTML to your system clipboard
3. Press **`p`** in the terminal where sucker is running
4. New items will be parsed and added to the download queue
5. Press **Ctrl+C** when done to exit

**Supported formats for clipboard:**
- HTML with links (`<a href="...">`)
- Plain text with URLs (one per line)

**Supported platforms:**
- **macOS**: Uses `pbpaste` for clipboard access
- **Linux**: Uses `xclip` or `wl-paste` (install one of these)

## Build

```shell
make build
```

## Testing

```shell
make test
```



