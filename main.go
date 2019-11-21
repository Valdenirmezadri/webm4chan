package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
)

var (
	ffmpegPath = flag.String("ff", "/usr/bin/ffmpeg", "ffmpeg location")
	extensao   = flag.String("ex", "webm", "extension of files you want: webm, mp4, avi")
	convert    = flag.String("c", "mp4", "convert to mp4")
	oneToOne   = flag.Bool("one", false, "download and converte one by one")
	url        = flag.String("url", "", "url of site")
	protocol   = flag.String("p", "http", "protocol, http, https")
	links      []string
	wg         sync.WaitGroup
)

func init() {
	flag.Parse()
}

func downloadAndConvert(link string, wg *sync.WaitGroup) {
	downloadFile(link, "./")
	p := strings.Split(link, "/")
	n := p[len(p)-1]
	n = strings.Replace(n, *extensao, "", 1)

	from := fmt.Sprintf("%v%v", n, *extensao)
	to := fmt.Sprintf("%v%v", n, *convert)
	if _, err := os.Stat(from); err != nil {
		os.Remove(from)

	}
	if _, err := os.Stat(to); err != nil {
		os.Remove(to)

	}
	cmd := exec.Command(*ffmpegPath, "-y", "-i", from, to)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		fmt.Println(fmt.Sprint(err) + ": " + stderr.String())
		//panic(1)
	}
	if _, err := os.Stat(from); err == nil {
		fmt.Printf("Removing %+v\n", from)
		os.Remove(from)

	}
	defer wg.Done()
}

func main() {
	links = scrapeLinks(*url)
	links = unique(links)
	wg.Add(len(links))
	for _, link := range links {
		if !*oneToOne {
			go downloadAndConvert(link, &wg)
		} else {
			downloadAndConvert(link, &wg)
		}
	}
	wg.Wait()
}

func scrapeLinks(url string) (list []string) {
	doc, err := goquery.NewDocument(url)

	if err != nil {
		panic(err)
	}

	doc.Find("a").Each(func(i int, s *goquery.Selection) {
		rawlinks, _ := s.Attr("href")
		links := strings.Fields(rawlinks)
		for _, link := range links {
			if strings.Contains(link, "webm") {
				list = append(list, fmt.Sprintf("%v:%v", *protocol, link))
			}
		}
	})
	return
}

func unique(intSlice []string) []string {
	keys := make(map[string]bool)
	list := []string{}
	for _, entry := range intSlice {
		if _, value := keys[entry]; !value {
			keys[entry] = true
			list = append(list, entry)
		}
	}
	return list
}

func printDownloadPercent(done chan int64, path string, total int64) {

	var stop bool = false

	for {
		select {
		case <-done:
			stop = true
		default:

			file, err := os.Open(path)
			if err != nil {
				log.Fatal(err)
			}

			fi, err := file.Stat()
			if err != nil {
				log.Fatal(err)
			}

			size := fi.Size()

			if size == 0 {
				size = 1
			}

			var percent float64 = float64(size) / float64(total) * 100

			fmt.Printf("%.0f", percent)
			fmt.Println("%")
		}

		if stop {
			break
		}

		time.Sleep(time.Second)
	}
}

func downloadFile(url string, dest string) {

	file := path.Base(url)

	log.Printf("Downloading file %s from %s\n", file, url)

	var path bytes.Buffer
	path.WriteString(dest)
	path.WriteString("/")
	path.WriteString(file)

	start := time.Now()

	out, err := os.Create(path.String())

	if err != nil {
		fmt.Println(path.String())
		panic(err)
	}

	defer out.Close()

	headResp, err := http.Head(url)

	if err != nil {
		panic(err)
	}

	defer headResp.Body.Close()

	size, err := strconv.Atoi(headResp.Header.Get("Content-Length"))

	if err != nil {
		panic(err)
	}

	done := make(chan int64)

	go printDownloadPercent(done, path.String(), int64(size))

	resp, err := http.Get(url)

	if err != nil {
		panic(err)
	}

	defer resp.Body.Close()

	n, err := io.Copy(out, resp.Body)

	if err != nil {
		panic(err)
	}

	done <- n

	elapsed := time.Since(start)
	log.Printf("Download completed in %s", elapsed)
}
