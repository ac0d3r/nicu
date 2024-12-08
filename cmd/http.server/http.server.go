package main

import (
	"encoding/base64"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"text/template"

	"github.com/ac0d3r/nicu/pkg/network"
)

const uiTmpl = `
<!DOCTYPE html>
<html>
<head>
    <title>File Manager</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; font-size: 12px; }
        h1 { color: #333; font-size: 18px; }
        .file-list { margin-top: 20px; padding: 0; list-style-type: none; }
        .file-item { 
            display: flex; 
            justify-content: space-between; 
            align-items: center; 
            padding: 8px 12px; 
            background-color: #f9f9f9; 
            margin-bottom: 4px; 
            border-radius: 4px;
        }
        .file-item:nth-child(even) { background-color: #f1f1f1; } /* 交替背景色 */
        .file-item a { text-decoration: none; color: #007BFF; font-weight: bold; }
        .file-item a:hover { text-decoration: underline; }
        .file-size { color: #555; font-size: 12px; }
        form { margin-top: 20px; }
        button { font-size: 14px; }
    </style>
</head>
<body>
    <h1>File Manager</h1>
	{{if .allowUpload}}
		<form action="/upload" method="post" enctype="multipart/form-data">
        	<input type="file" name="file">
        	<button type="submit">Upload</button>
    	</form>
	{{end}}
    <h2>Files:</h2>
    <ul class="file-list">
        {{range .files}}
        <li class="file-item">
            <a href="/download/{{.Name}}" download>{{.Name}}</a>
            <span class="file-size">{{.Size}}</span>
        </li>
        {{end}}
    </ul>
</body>
</html>`

var (
	flagHelp = flag.Bool("h", false, "Shows usage options.")
	flagHost = flag.String("l", ":8080", "listen host")
	flagDir  = flag.String("d", "./", "load directory")
	flagFile = flag.String("f", "", "load single file")
	flagKey  = flag.String("a", "admin:admin", "authentication key e.g.(admin:admin)")
)

func banner() {
	t := `
   __   __  __                                 
  / /  / /_/ /____    ___ ___ _____  _____ ____
 / _ \/ __/ __/ _ \_ (_-</ -_) __/ |/ / -_) __/
/_//_/\__/\__/ .__(_)___/\__/_/  |___/\__/_/   
            /_/                                
`
	fmt.Println(t)
}

func main() {
	banner()
	flag.Parse()
	if *flagHelp {
		fmt.Printf("Usage: http.server [options]\n\n")
		flag.PrintDefaults()
		return
	}

	if len(strings.Split(*flagHost, ":")[0]) == 0 {
		fmt.Printf("listen on http://localhost%s\n", *flagHost)
		if ipnets, err := network.GetLocalIPV4Net(); err == nil && len(ipnets) >= 1 {
			fmt.Printf("listen on http://%s%s\n", ipnets[0].IP, *flagHost)
		}
	} else {
		fmt.Printf("listen on http://%s\n", *flagHost)
	}

	dir := "./"
	readonlys := make([]string, 0)
	if *flagFile != "" {
		readonlys = append(readonlys, *flagFile)
	} else if *flagDir != "" {
		dir = *flagDir
	}

	http.HandleFunc("/", basicAuth(func(w http.ResponseWriter, r *http.Request) {
		files, err := listFiles(dir, readonlys...)
		if err != nil {
			http.Error(w, "Failed to list files", http.StatusInternalServerError)
			return
		}
		t := template.Must(template.New("index").Parse(uiTmpl))
		t.Execute(w, map[string]any{
			"files":       files,
			"allowUpload": len(readonlys) == 0,
		})
	}))

	http.HandleFunc("/upload", basicAuth(func(w http.ResponseWriter, r *http.Request) {
		if len(readonlys) > 0 {
			http.Error(w, "Read Only", http.StatusBadRequest)
			return
		}

		if r.Method != http.MethodPost {
			http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
			return
		}

		file, header, err := r.FormFile("file")
		if err != nil {
			http.Error(w, "Failed to read file", http.StatusBadRequest)
			return
		}
		defer file.Close()

		filePath := filepath.Join(dir, header.Filename)
		out, err := os.Create(filePath)
		if err != nil {
			http.Error(w, "Failed to save file", http.StatusInternalServerError)
			return
		}
		defer out.Close()

		if _, err = out.ReadFrom(file); err != nil {
			http.Error(w, "Failed to save file", http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, "/", http.StatusSeeOther)
	}))

	http.HandleFunc("/download/", basicAuth(func(w http.ResponseWriter, r *http.Request) {
		filename := filepath.Base(r.URL.Path)
		if len(readonlys) > 0 {
			if !slices.Contains(readonlys, filename) {
				http.Error(w, "Read Only", http.StatusBadRequest)
				return
			}
		}

		filePath := filepath.Join(dir, filename)

		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			http.Error(w, "File not found", http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Disposition", "attachment; filename="+filename)
		http.ServeFile(w, r, filePath)
	}))

	log.Fatal(http.ListenAndServe(*flagHost, nil))
}

type FileInfo struct {
	Name string
	Size string
}

func listFiles(dir string, filters ...string) ([]FileInfo, error) {
	files := []FileInfo{}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			info, err := entry.Info()
			if err != nil {
				return nil, err
			}

			if len(filters) > 0 {
				if !slices.Contains(filters, entry.Name()) {
					continue
				}
			}

			size := formatFileSize(info.Size())
			files = append(files, FileInfo{
				Name: entry.Name(),
				Size: size,
			})
		}
	}
	return files, nil
}

func formatFileSize(size int64) string {
	if size < 1024 {
		return fmt.Sprintf("%d B", size)
	} else if size < 1024*1024 {
		return fmt.Sprintf("%.2f KB", float64(size)/1024)
	} else {
		return fmt.Sprintf("%.2f MB", float64(size)/(1024*1024))
	}
}

func basicAuth(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth == "" || !checkAuth(auth) {
			w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		handler(w, r)
	}
}

// "Basic <base64encoded(username:password)>"
func checkAuth(authHeader string) bool {
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || parts[0] != "Basic" {
		return false
	}

	payload, err := base64.StdEncoding.DecodeString(parts[1])
	if err != nil {
		return false
	}

	credentials := strings.SplitN(string(payload), ":", 2)
	if len(credentials) != 2 {
		return false
	}

	unamepass := strings.SplitN(*flagKey, ":", 2)
	if len(credentials) != 2 {
		return false
	}

	return credentials[0] == unamepass[0] && credentials[1] == unamepass[1]
}
