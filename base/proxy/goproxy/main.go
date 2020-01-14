package main

import (
	"bufio"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
)

var host = os.Getenv("PROXY_HOST")
var homedest = os.Getenv("HOMEDEST")
var secret = os.Getenv("SECRET")

var passThroughProxy *httputil.ReverseProxy
var reverseProxy *httputil.ReverseProxy
var proxies = make(map[string]map[string]*url.URL)

func init() {
	passThroughProxy = &httputil.ReverseProxy{
		Director:       passThroughDirector,
		ModifyResponse: modifyResponse,
	}
	reverseProxy = &httputil.ReverseProxy{
		Director:       reverseDirector,
		ModifyResponse: modifyResponse,
	}

	log.Printf("homedest: %s", homedest)
}

func main() {
	fileInfos, err := ioutil.ReadDir("/config")
	if err != nil {
		log.Fatal(err)
	}

	for _, fileInfo := range fileInfos {
		app := fileInfo.Name()
		file, err := os.Open("/config/" + app)
		if err != nil {
			log.Fatal(err)
		}
		defer file.Close()

		proxies[app] = make(map[string]*url.URL)
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			split := strings.Split(scanner.Text(), "=")
			if len(split) == 2 {
				defineProxy(app, split[0], split[1])
			}
		}

		if err := scanner.Err(); err != nil {
			log.Fatal(err)
		}
	}

	// Redirect http to https
	go http.ListenAndServe(":80", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		url := r.URL
		url.Host = r.Host
		url.Scheme = "https"
		http.Redirect(w, r, url.String(), http.StatusMovedPermanently)
	}))

	crt := "/certificates/" + host + ".crt"
	key := "/certificates/" + host + ".key"
	log.Fatal(http.ListenAndServeTLS(":443", crt, key, http.HandlerFunc(handler)))
}

func handler(w http.ResponseWriter, r *http.Request) {
	log.Printf("%s %s %s", r.RemoteAddr, r.Method, r.RequestURI)
	if r.Method == "OPTIONS" {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, PUT, POST, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "SOAPAction, X-Requested-With, Origin, Content-Type, Authorization, Accept")
	} else if r.URL.Path == "/" {
		if r.URL.Query().Get("url") == "" && homedest != "" {
			http.Redirect(w, r, homedest, http.StatusFound)
		} else {
			passThroughProxy.ServeHTTP(w, r)
		}
	} else {
		reverse(w, r)
	}
}

func defineProxy(app, key, target_ string) {
	target, _ := url.Parse(target_)
	if target.Path == "" {
		target.Path = "/"
	}
	key = "/" + key + "/"
	proxies[app][key] = target
	log.Printf("reversing: %s%s -> %v", app, key, target)
}

func reverse(w http.ResponseWriter, r *http.Request) {
	path := "/"
	requestParts := strings.Split(r.URL.Path, "/")
	app := requestParts[1]
	if _, ok := proxies[app]; ok {
		if len(requestParts) > 2 {
			path += strings.SplitN(r.URL.Path, "/", 3)[2] // alles na de tweede slash
		}
		// log.Printf("app=%s path=%s", app, path)
	} else {
		referer, _ := url.Parse(r.Referer() + "/") //  + "/" to cater for empty Referer
		app = strings.Split(referer.Path, "/")[1]
		path = r.URL.Path
		// log.Printf("referer app=%s path=%s", app, path)
	}
	for key, target := range proxies[app] {
		if path+"/" == key {
			path = path + "/"
		}
		if strings.HasPrefix(path, key) {
			r.URL.Scheme = target.Scheme
			r.URL.Host = target.Host
			r.URL.Path = target.Path + strings.SplitN(path, "/", 3)[2] // alles na de tweede slash
			r.Host = target.Host
			reverseProxy.ServeHTTP(w, r)
			return
		}
	}
	http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
}

func reverseDirector(r *http.Request) {
	log.Printf("%s %s reverse %v", r.RemoteAddr, r.Method, r.URL)
	if _, ok := r.Header["User-Agent"]; !ok {
		// explicitly disable User-Agent so it's not set to default value
		r.Header.Set("User-Agent", "")
	}
}

func passThroughDirector(r *http.Request) {
	if target, err := url.Parse(r.FormValue("url")); err != nil {
		log.Printf("%+v", err)
	} else {
		query := r.URL.Query()
		query.Del("url")
		target.RawQuery = query.Encode()
		r.URL = target
		r.Host = target.Host
		log.Printf("%s %s passthrough %v", r.RemoteAddr, r.Method, r.URL)
	}
}

func modifyResponse(r *http.Response) error {
	r.Header.Set("Access-Control-Allow-Origin", "*")
	// Deze twee hieronder zouden eigenlijk niet nodig moeten zijn, maar het blijkt er wel beter van te worden..
	r.Header.Set("Access-Control-Allow-Methods", "GET, PUT, POST, DELETE, OPTIONS")
	r.Header.Set("Access-Control-Allow-Headers", "SOAPAction, X-Requested-With, Origin, Content-Type, Authorization, Accept")
	return nil
}
