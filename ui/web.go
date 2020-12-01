package ui

import (
	"bufio"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"time"

	"github.com/murakmii/gotermda/pty"
	"github.com/murakmii/gotermda/shell"
)

type (
	WebUI struct {
		server     *http.Server
		nextTermId int
		opened     map[int]*openedTerminal
	}

	openedTerminal struct {
		sh      *shell.Shell
		master  *os.File
		slave   *os.File
		buffer  []rune
		updated chan struct{}
	}

	openedResponse struct {
		TerminalId int `json:"terminal_id"`
	}
)

var (
	writePath = regexp.MustCompile("/write/(\\d+)")
	readPath  = regexp.MustCompile("/read/(\\d+)")
)

func NewWebUI() *WebUI {
	return &WebUI{
		nextTermId: 1,
		opened:     make(map[int]*openedTerminal, 0),
	}
}

func (webUI *WebUI) ListenAndServe(addr string) error {
	webUI.server = &http.Server{
		Addr:    addr,
		Handler: http.HandlerFunc(webUI.requestHandler),
	}

	return webUI.server.ListenAndServe()
}

func (webUI *WebUI) requestHandler(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/", "/style.css", "/gotermda.js":
		handleResource(w, r)

	case "/open":
		webUI.handleOpen(w, r)

	default:
		pathBytes := []byte(r.URL.Path)
		if match := writePath.FindSubmatch(pathBytes); len(match) > 1 {
			webUI.handleWrite(w, r, string(match[1]))
		} else if match := readPath.FindSubmatch(pathBytes); len(match) > 1 {
			webUI.handleRead(w, r, string(match[1]))
		}
	}
}

func handleResource(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.NotFound(w, r)
		return
	}

	resource := r.URL.Path
	if resource == "/" {
		resource = "index.html"
	}
	http.ServeFile(w, r, filepath.Join("./resource", resource))
}

func (webUI *WebUI) handleOpen(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.NotFound(w, r)
		return
	}

	master, slave, err := pty.Open()
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to open terminal: %s", err), 500)
		return
	}

	sh, err := shell.Start("/bin/bash", slave)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to start shell: %s", err), 500)
		return
	}

	opened := &openedTerminal{
		sh:      sh,
		master:  master,
		slave:   slave,
		buffer:  make([]rune, 0),
		updated: make(chan struct{}),
	}

	webUI.opened[webUI.nextTermId] = opened
	response, _ := json.Marshal(&openedResponse{TerminalId: webUI.nextTermId})
	webUI.nextTermId++

	go func() {
		reader := bufio.NewReader(master)
		for {
			r, _, err := reader.ReadRune()
			if err != nil {
				// TODO: logging
			}
			opened.buffer = append(opened.buffer, r)
		}
	}()

	w.WriteHeader(http.StatusOK)
	w.Write(response)
}

func (webUI *WebUI) handleWrite(w http.ResponseWriter, r *http.Request, termIdStr string) {
	if r.Method != http.MethodPut {
		http.NotFound(w, r)
		return
	}

	opened := webUI.findOpenedTerminal(w, r, termIdStr)
	if opened == nil {
		return
	}

	if _, err := io.Copy(opened.master, r.Body); err != nil {
		http.Error(w, fmt.Sprintf("failed to write to terminal: %s", err), 500)
		return
	}

	w.WriteHeader(200)
}

func (webUI *WebUI) handleRead(w http.ResponseWriter, r *http.Request, termIdStr string) {
	if r.Method != http.MethodGet {
		http.NotFound(w, r)
		return
	}

	opened := webUI.findOpenedTerminal(w, r, termIdStr)
	if opened == nil {
		return
	}

	flusher, _ := w.(http.Flusher)
	closed := r.Context().Done()

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(200)

	fmt.Fprintf(w, "data: %s\n\n", base64.RawStdEncoding.EncodeToString([]byte(string(opened.buffer))))
	flusher.Flush()

	for {
		select {
		case <-closed:
		// TODO: close terminal

		case <-time.After(1 * time.Second):
			fmt.Fprintf(w, "data: %s\n\n", base64.RawStdEncoding.EncodeToString([]byte(string(opened.buffer))))
			flusher.Flush()
		}
	}
}

func (webUI *WebUI) findOpenedTerminal(w http.ResponseWriter, r *http.Request, termIdStr string) *openedTerminal {
	termId, err := strconv.ParseInt(termIdStr, 10, 31)
	if err != nil {
		http.Error(w, fmt.Sprintf("invalid term id: %s", err), 500)
		return nil
	}

	opened, exists := webUI.opened[int(termId)]
	if !exists {
		http.NotFound(w, r)
		return nil
	}

	return opened
}
