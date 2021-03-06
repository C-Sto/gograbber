package libgograbber

import (
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/pmezard/go-difflib/difflib"
)

// // checks to see whether host is http/s or other scheme.
// // Returns error if endpoint is not a valid webserver. - This just
// func Prefetch(host Host, debug bool, jitter int, protocols StringSet) (h Host, err error) {
// Removed becuase it just kept breaking... 😔 🤔
// }

func HTTPGetter(wg *sync.WaitGroup, host Host, debug bool, Jitter int, soft404Detection bool, statusCodesIgn IntSet, Ratio float64, path string, results chan Host, threads chan struct{}, ProjectName string, responseDirectory string, writeChan chan []byte, followRedirects bool) {
	defer func() {
		<-threads
		wg.Done()
	}()

	if strings.HasPrefix(path, "/") && len(path) > 0 {
		path = path[1:] // strip preceding '/' char
	}
	Url := fmt.Sprintf("%v://%v:%v/%v", host.Protocol, host.HostAddr, host.Port, path)
	if debug {
		Debug.Printf("Trying URL: %v\n", Url)
	}
	ApplyJitter(Jitter)

	var err error
	nextUrl := Url
	var i int
	for i < 5 { // number of times to follow redirect

		host.HTTPReq, host.HTTPResp, err = host.makeHTTPRequest(nextUrl)
		if err != nil {
			return
		}
		if statusCodesIgn.Contains(host.HTTPResp.StatusCode) {
			host.HTTPResp.Body.Close()
			return
		}
		if host.HTTPResp.StatusCode >= 300 && host.HTTPResp.StatusCode < 400 && followRedirects {
			host.HTTPResp.Body.Close()
			x, err := host.HTTPResp.Location()
			if err == nil {
				nextUrl = x.String()
			} else {
				break
			}
		} else {
			defer host.HTTPResp.Body.Close()
			Url = nextUrl
			break
		}
	}
	if soft404Detection && path != "" && host.Soft404RandomPageContents != nil {
		soft404Ratio := detectSoft404(host.HTTPResp, host.Soft404RandomPageContents)
		if soft404Ratio > Ratio {
			if debug {
				Debug.Printf("[%v] is very similar to [%v] (%v match)\n", y.Sprintf("%s", Url), y.Sprintf("%s", host.Soft404RandomURL), y.Sprintf("%.4f%%", (soft404Ratio*100)))
			}
			return
		}
	}
	buf, err := ioutil.ReadAll(host.HTTPResp.Body)

	if host.HostHeader != "" {
		Good.Printf("%v - %v [%v bytes] (HostHeader: %v)\n", Url, g.Sprintf("%d", host.HTTPResp.StatusCode), len(buf), host.HostHeader)

	} else {
		Good.Printf("%v - %v [%v bytes]\n", Url, g.Sprintf("%d", host.HTTPResp.StatusCode), len(buf))
	}
	currTime := GetTimeString()

	var responseFilename string
	if ProjectName != "" {
		responseFilename = fmt.Sprintf("%v/%v_%v-%v_%v.html", responseDirectory, strings.ToLower(SanitiseFilename(ProjectName)), SanitiseFilename(Url), currTime, rand.Int63())
	} else {
		responseFilename = fmt.Sprintf("%v/%v-%v_%v.html", responseDirectory, SanitiseFilename(Url), currTime, rand.Int63())
	}
	file, err := os.Create(responseFilename)
	if err != nil {
		Error.Printf("%v\n", err)
	}
	if err != nil {
		Error.Printf("%v\n", err)
	} else {
		if len(buf) > 0 {
			file.Write(buf)
			host.ResponseBodyFilename = responseFilename
		} else {
			_ = os.Remove(responseFilename)
		}
	}
	host.Path = path
	writeChan <- []byte(fmt.Sprintf("%v\n", Url))
	results <- host
}

func detectSoft404(resp *http.Response, randRespData []string) (ratio float64) {
	// defer resp.Body.Close()
	diff := difflib.SequenceMatcher{}
	responseData, _ := ioutil.ReadAll(resp.Body)
	diff.SetSeqs(strings.Split(string(responseData), " "), randRespData)
	ratio = diff.Ratio()
	return ratio
}
