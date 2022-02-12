package main

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hashicorp/go-version"
)

var alpineTodos map[string][]string
var apkGithubMapping = map[string]string{
	"coredns":    "coredns/coredns",
	"conntracct": "ti-mo/conntracct",
	"corerad":    "mdlayher/corerad",
	"mage":       "magefile/mage",
}

func main() {
	ticker := time.NewTicker(1 * time.Hour)
	done := make(chan struct{})
	go func() {
		for {
			alpineTodos = make(map[string][]string)
			doAlpine()
			select {
			case <-done:
				return
			case <-ticker.C:
				continue
			}
		}
	}()

	r := gin.Default()
	r.LoadHTMLGlob("templates/*.tmpl")
	r.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.tmpl", gin.H{
			"title":       "markpash.todo",
			"alpineTasks": alpineTodos,
		})
	})
	r.Run()
	done <- struct{}{}
}

func doAlpine() {
	mainIdx, err := fetchAPKIndex(edgeMain)
	if err != nil {
		panic(err)
	}

	communityIdx, err := fetchAPKIndex(edgeCommunity)
	if err != nil {
		panic(err)
	}

	testingIdx, err := fetchAPKIndex(edgeTesting)
	if err != nil {
		panic(err)
	}

	myPkgs := filterForMaintainer("mark@markpash.me", mainIdx, communityIdx, testingIdx)

	for name, ap := range myPkgs {
		ghRepo, ok := apkGithubMapping[name]
		if !ok {
			continue
		}

		apkVer, err := version.NewVersion(ap.version)
		if err != nil {
			panic(err)
		}
		ghLatest, err := getLatestReleaseVersion(ghRepo)
		if err != nil {
			panic(err)
		}

		ghVer, err := version.NewVersion(ghLatest)
		if err != nil {
			panic(err)
		}

		if ghVer.GreaterThan(apkVer) {
			alpineTodos[name] = []string{ghVer.String(), apkVer.String()}
		}
	}
}
