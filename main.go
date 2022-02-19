package main

import (
	"context"
	"embed"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hashicorp/go-version"
)

//go:embed templates/*
var templates embed.FS

var alpineTodos map[string][]string = make(map[string][]string)
var apkGithubMapping = map[string]string{
	"coredns":    "coredns/coredns",
	"conntracct": "ti-mo/conntracct",
	"corerad":    "mdlayher/corerad",
	"mage":       "magefile/mage",
}

func main() {
	if err := run(); err != nil {
		fmt.Fprint(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	go func() {
		defer cancel()

		r := gin.Default()
		r.SetHTMLTemplate(template.Must(template.New("").ParseFS(templates, "templates/*.tmpl")))
		r.GET("/", func(c *gin.Context) {
			c.HTML(http.StatusOK, "index.tmpl", gin.H{
				"title":       "markpash.todo",
				"alpineTasks": alpineTodos,
			})
		})
		r.Run()
	}()

	ticker := time.NewTicker(1 * time.Hour)
	for {
		if err := doAlpine(ctx); err != nil {
			return err
		}

		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			continue
		}
	}
}

func doAlpine(ctx context.Context) error {
	// reset the global map
	alpineTodos = make(map[string][]string)

	client := http.Client{
		Timeout: 10 * time.Second,
	}

	mainIdx, err := fetchAPKIndex(ctx, client, edgeMain)
	if err != nil {
		return err
	}

	communityIdx, err := fetchAPKIndex(ctx, client, edgeCommunity)
	if err != nil {
		return err
	}

	testingIdx, err := fetchAPKIndex(ctx, client, edgeTesting)
	if err != nil {
		return err
	}

	myPkgs := filterForMaintainer("mark@markpash.me", mainIdx, communityIdx, testingIdx)

	for name, ap := range myPkgs {
		ghRepo, ok := apkGithubMapping[name]
		if !ok {
			continue
		}

		apkVer, err := version.NewVersion(ap.version)
		if err != nil {
			return err
		}
		ghLatest, err := getLatestReleaseVersion(ctx, client, ghRepo)
		if err != nil {
			return err
		}

		ghVer, err := version.NewVersion(ghLatest)
		if err != nil {
			return err
		}

		if ghVer.GreaterThan(apkVer) {
			alpineTodos[name] = []string{ghVer.String(), apkVer.String()}
		}
	}

	return nil
}
