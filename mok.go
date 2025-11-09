// sometimes when writing go i feel like "the third rob"
//
// https://en.wikipedia.org/wiki/Rob_Pike
// https://github.com/empijei
//
// P.S. https://youtu.be/WHqbqzqeskw
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"log"
	"strconv"
	"strings"

	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
)

const indexTemplate = `
<!doctype html>
<html lang="en">
    <head>
        <meta charset="UTF-8" />
        <meta
            name="viewport"
            content="width=device-width, user-scalable=no, initial-scale=1.0, maximum-scale=1.0, minimum-scale=1.0"
        />
        <title>mok</title>
    </head>
    <body>
        <h1>mok</h1>
        available endpoints:
        <ul>
            {{range .}}
            <li><a href="{{.URLPath}}">{{.FilePath}}</a></li>
            {{end}}
        </ul>
    </body>
</html>
`

var usage = `
  usage: mok [options] <files.json>

  files can be local or remote (api endpoints):
    remote: URI must start with http:// or https://
    local: passing directories is not supported, use glob instead.

  additionally mok reads json from stdin, try it with 'echo '{"k": "v"}' | mok'

  options:
    -p <port>           specify the port to listen on
    -s <json string>    specify the json string to serve (on /)
    -v                  verbose output

`

var (
	portPtr    = flag.Int("p", 9172, "specify the port to listen on")
	jsonStrPtr = flag.String("s", "", "specify the json string to serve")
	verbosePtr = flag.Bool("v", false, "verbose output")
)

func errAndExit(msg string) {
	fmt.Fprintf(os.Stderr, "error: %s\n\n", msg)
	os.Exit(1)
}

func main() {
	flag.Usage = func() {
		fmt.Fprint(os.Stderr, usage)
	}
	flag.Parse()

	directInput := getDirectInput()

	if flag.NArg() < 1 && len(directInput) == 0 {
		errAndExit("no file specified")
	}
	// mok receives exactly what the shell passes.
	//   ./mok testdata/*.json
	// shells expand the glob before execution, so the program sees:
	//   ./mok testdata/a.json testdata/b.json ...
	// curious rabbits: https://man7.org/linux/man-pages/man7/glob.7.html
	files := processFileArgs(flag.Args())

	setupHandlers(directInput, files)

	if len(directInput) == 0 {
		printSummary(*portPtr, files)
	} else {
		fmt.Printf("mok is serving direct input on http://localhost:%d/\n", *portPtr)
	}

	if err := http.ListenAndServe(":"+strconv.Itoa(*portPtr), nil); err != nil {
		errAndExit("http: " + err.Error())
	}
}

type MokFile struct {
	FilePath string
	URLPath  string
}

func downloadJSON(_url string) (string, error) {
	logInfo(fmt.Sprintf("downloading: %q", _url))
	u, err := url.Parse(_url)
	if err != nil {
		return "", fmt.Errorf("parse URL: %w", err)
	}
	tempFile, err := os.CreateTemp("", fmt.Sprintf("mok-%s.*.json", u.Host))
	if err != nil {
		return "", fmt.Errorf("create temp file: %w", err)
	}
	defer tempFile.Close()
	logInfo(fmt.Sprintf("creating temp file: %q", tempFile.Name()))

	resp, err := http.Get(_url)
	if err != nil {
		return "", fmt.Errorf("download: %w", err)
	}
	defer resp.Body.Close()

	if resp.Header.Get("Content-Type") != "application/json" {
		return "", fmt.Errorf("unexpected content type for %q: %s", _url, resp.Header.Get("Content-Type"))
	}

	if resp.StatusCode != http.StatusOK {
		logInfo(fmt.Sprintf("failed to download file from: %q", _url))
		return "", fmt.Errorf("download failed: %s", resp.Status)
	}

	if _, err := io.Copy(tempFile, resp.Body); err != nil {
		return "", fmt.Errorf("save: %w", err)
	}

	logInfo(fmt.Sprintf("succesfully downloaded file %q to %q", _url, tempFile.Name()))
	return tempFile.Name(), nil
}

func printSummary(port int, files []MokFile) {
	baseURL := fmt.Sprintf("http://localhost:%d", port)

	fmt.Printf("  mok is listening at %s\n\n", baseURL)
	fmt.Println("  available endpoints:")

	maxURLLen := 0
	for _, file := range files {
		urlLen := len("GET " + file.URLPath)
		if urlLen > maxURLLen {
			maxURLLen = urlLen
		}
	}

	for _, file := range files {
		url := file.URLPath
		source := fmt.Sprintf("(%s)", file.FilePath)

		padding := maxURLLen - len(" "+url)
		spaces := strings.Repeat(" ", padding)

		fmt.Printf("   %s%s  %s\n", url, spaces, source)
	}
}

func logInfo(msg string) {
	if *verbosePtr {
		log.Println(msg)
	}
}

func getDirectInput() []byte {
	// stdin first
	fi, err := os.Stdin.Stat()
	if err != nil {
		errAndExit("cannot read direct input: " + err.Error())
	}

	if size := fi.Size(); size > 0 {
		directInput := make([]byte, size)
		if _, err := os.Stdin.Read(directInput); err != nil {
			errAndExit("cannot read direct input: " + err.Error())
		}
		return directInput
	}

	// then `-s` flag
	if *jsonStrPtr != "" {
		return []byte(*jsonStrPtr)
	}

	return nil
}

func processFileArgs(args []string) []MokFile {
	seen := make(map[string]struct{})
	var files []MokFile

	for _, arg := range args {
		filePath, err := resolveFile(arg)
		if err != nil {
			errAndExit(err.Error())
		}

		if _, exists := seen[filePath]; exists {
			continue
		}

		seen[filePath] = struct{}{}
		files = append(files, MokFile{
			FilePath: filePath,
			URLPath:  "/" + filepath.Base(filePath),
		})
	}

	return files
}

func resolveFile(arg string) (string, error) {
	// remote
	if strings.HasPrefix(arg, "http://") || strings.HasPrefix(arg, "https://") {
		file, err := downloadJSON(arg)
		if err != nil {
			return "", fmt.Errorf("downloading remote file: %w", err)
		}
		return file, nil
	}

	// local
	info, err := os.Stat(arg)
	if err != nil {
		return "", fmt.Errorf("checking file: %w", err)
	}
	if info.IsDir() {
		return "", fmt.Errorf("argument is a directory: %s", arg)
	}

	return arg, nil
}

func setupHandlers(directInput []byte, files []MokFile) {
	tmpl := template.Must(template.New("").Parse(indexTemplate))

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if len(directInput) > 0 {
			serveDirectInput(w, directInput)
			return
		}

		if r.Header.Get("Accept") == "application/json" {
			json.NewEncoder(w).Encode(files)
			return
		}

		tmpl.Execute(w, files)
	})

	for _, f := range files {
		_, fileName := filepath.Split(f.FilePath)

		http.HandleFunc("/"+fileName, func(w http.ResponseWriter, r *http.Request) {
			http.ServeFile(w, r, f.FilePath)
		})
	}
}

func serveDirectInput(w http.ResponseWriter, input []byte) {
	var dat map[string]any
	if err := json.Unmarshal(input, &dat); err != nil {
		http.Error(w, "Invalid JSON: "+err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(dat)
}
