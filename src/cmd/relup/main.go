package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"strings"
)

func main() {
	log.SetFlags(0)
	if len(os.Args) != 4 {
		log.Fatal("Usage: relup <user/repo> <tagname> <assetfile>\n  i.e. relup calmh/relup v1.0.0 relup-binary.tar.gz")
	}
	repo := os.Args[1]
	tag := os.Args[2]
	file := os.Args[3]
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		log.Fatal("Please export GITHUB_TOKEN=\"<your token here>\"")
	}

	req, err := http.NewRequest("GET", fmt.Sprintf("https://api.github.com/repos/%s/releases", repo), nil)
	req.Header.Set("Authorization", "token "+token)
	if err != nil {
		log.Fatal(err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		bs, _ := ioutil.ReadAll(resp.Body)
		log.Fatal(resp.Status + ":\n" + string(bs))
	}

	var res []map[string]interface{}
	dec := json.NewDecoder(resp.Body)
	dec.UseNumber()
	err = dec.Decode(&res)
	if err != nil {
		log.Fatal(err)
	}
	resp.Body.Close()

	var uploadURL string
	for _, rel := range res {
		if rel["tag_name"] == tag {
			uploadURL = rel["upload_url"].(string)
			break
		}
	}

	if uploadURL == "" {
		log.Fatalln("Found no release with that tag")
	}

	fd, err := os.Open(file)
	if err != nil {
		log.Fatal(err)
	}
	fi, _ := fd.Stat()

	info, _ := fd.Stat()
	size := info.Size()
	progr := &progressWriter{
		expected: size,
	}
	tee := io.TeeReader(fd, progr)

	log.Println("Uploading", path.Base(file))
	url := strings.Replace(uploadURL, "{?name,label}", "?name="+path.Base(file), 1)
	req, err = http.NewRequest("POST", url, tee)
	if err != nil {
		log.Fatal(err)
	}

	req.ContentLength = fi.Size()
	req.Header.Set("Content-Type", "application/octet-stream")
	req.Header.Set("Authorization", "token "+token)

	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		log.Fatal(err)
	}

	progr.Close()
	log.Println(resp.Status)
}

type progressWriter struct {
	expected int64
	current  int64
}

func (p *progressWriter) Write(data []byte) (int, error) {
	p.current += int64(len(data))
	p.printProgress()
	return len(data), nil
}

func (p *progressWriter) Close() error {
	fmt.Println()
	return nil
}

func (p *progressWriter) printProgress() {
	pct := float64(p.current) / float64(p.expected) * 100
	fmt.Printf("\r%d / %d (%5.01f%%)", p.current, p.expected, pct)
}
