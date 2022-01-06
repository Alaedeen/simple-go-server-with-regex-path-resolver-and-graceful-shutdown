package main

import (
	"bufio"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"regexp"
	"strings"

	"github.com/braintree/manners"
)

func main() {
	rr := newPathResolver()

	ch := make(chan os.Signal)
	signal.Notify(ch, os.Interrupt, os.Kill)
	go listenForShutdown(ch)

	rr.Add("GET /read", readFile)
	rr.Add("POST /add(/?[A-Za-z0-9]*)?", createFile)
	rr.Add("(PATCH|PUT) /update(/?[A-Za-z0-9]*)?", updateFile)
	manners.ListenAndServe(":8080", rr)
}

func newPathResolver() *regexResolver {
	return &regexResolver{
		handlers: make(map[string]http.HandlerFunc),
		cache:    make(map[string]*regexp.Regexp),
	}
}

type regexResolver struct {
	handlers map[string]http.HandlerFunc
	cache    map[string]*regexp.Regexp
}

func (r *regexResolver) Add(regex string, handler http.HandlerFunc) {
	r.handlers[regex] = handler
	cache, _ := regexp.Compile(regex)
	r.cache[regex] = cache
}

func (r *regexResolver) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	check := req.Method + " " + req.URL.Path
	for pattern, handlerFunc := range r.handlers {
		if r.cache[pattern].MatchString(check) == true {
			handlerFunc(res, req)
			return
		}
	}

	http.NotFound(res, req)
}

func getFileName(req *http.Request) (string, error) {
	path := req.URL.Path
	parts := strings.Split(path, "/")
	fileName := ""
	if len(parts) > 2 {
		fileName = parts[2]
	}
	if fileName == "" {
		return "", fmt.Errorf("missing file name")
	}
	return fileName, nil
}

func getFileContent(req *http.Request) (string, error) {
	query := req.URL.Query()
	text := query.Get("text")
	if text == "" {
		return "", fmt.Errorf("missing file content")
	}
	return text, nil
}

func readFile(res http.ResponseWriter, req *http.Request) {
	fileName, err := getFileName(req)
	if err != nil {
		res.WriteHeader(http.StatusBadRequest)
		res.Write([]byte(err.Error()))
		return
	}

	f, err := os.Open("./files/" + fileName)
	if err != nil {
		res.WriteHeader(http.StatusBadRequest)
		res.Write([]byte(err.Error()))
		return
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	var text string 
	for scanner.Scan() {
		text +=  scanner.Text() +"\n"
	}
	fmt.Fprint(res, text)
}

func createFile(res http.ResponseWriter, req *http.Request) {
	fileName, err := getFileName(req)
	if err != nil {
		res.WriteHeader(http.StatusBadRequest)
		res.Write([]byte(err.Error()))
		return
	}

	text, err := getFileContent(req)
	if err != nil {
		res.WriteHeader(http.StatusBadRequest)
		res.Write([]byte(err.Error()))
		return
	}
	_, err = os.Open("./files/" + fileName)
	if err == nil {
		res.WriteHeader(http.StatusInternalServerError)
		res.Write([]byte("file exists already"))
		return
	}

	f, err := os.Create("./files/" + fileName)
	if err != nil {
		res.WriteHeader(http.StatusInternalServerError)
		res.Write([]byte(err.Error()))
		return
	}
	defer f.Close()
	_, err = f.WriteString(text + "\n")
	if err != nil {
		res.WriteHeader(http.StatusInternalServerError)
		res.Write([]byte(err.Error()))
		return
	}

	fmt.Fprint(res, fileName, " created successfully")

}

func updateFile(res http.ResponseWriter, req *http.Request) {
	fileName, err := getFileName(req)
	if err != nil {
		res.WriteHeader(http.StatusBadRequest)
		res.Write([]byte(err.Error()))
		return
	}

	text, err := getFileContent(req)
	if err != nil {
		res.WriteHeader(http.StatusBadRequest)
		res.Write([]byte(err.Error()))
		return
	}
	f, err := os.OpenFile("./files/"+fileName, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		res.WriteHeader(http.StatusInternalServerError)
		res.Write([]byte(err.Error()))
		return
	}

	defer f.Close()
	_, err = f.WriteString(text + "\n")
	if err != nil {
		res.WriteHeader(http.StatusInternalServerError)
		res.Write([]byte(err.Error()))
		return
	}

	fmt.Fprint(res, fileName, " updated successfully")
}

func listenForShutdown(ch <-chan os.Signal) {
	<-ch
	manners.Close()
}
