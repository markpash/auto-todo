package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"io"
	"net/http"
	"regexp"
	"strings"
)

const edgeMain string = "http://dl-cdn.alpinelinux.org/alpine/edge/main/x86_64/APKINDEX.tar.gz"
const edgeCommunity string = "http://dl-cdn.alpinelinux.org/alpine/edge/community/x86_64/APKINDEX.tar.gz"
const edgeTesting string = "http://dl-cdn.alpinelinux.org/alpine/edge/testing/x86_64/APKINDEX.tar.gz"

var maintainerRegex = regexp.MustCompile(`(.*)\s<(.*)>`)
var apkVerRegex = regexp.MustCompile(`(.*)\-(.*)`)

type apkPackage struct {
	name            string
	maintainerName  string
	maintainerEmail string
	version         string
	revision        string
}

type apkIndex struct {
	desc  string
	index map[string]apkPackage
	sig   []byte
}

func fetchAPKIndex(ctx context.Context, client http.Client, url string) (apkIndex, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return apkIndex{}, err
	}

	index, err := client.Do(req)
	if err != nil {
		return apkIndex{}, err
	}
	defer index.Body.Close()

	gzReader, err := gzip.NewReader(index.Body)
	if err != nil {
		return apkIndex{}, err
	}

	var ret apkIndex

	tarReader := tar.NewReader(gzReader)
	for {
		hdr, err := tarReader.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return apkIndex{}, err
		}

		// Handle the .SIGN file
		if strings.HasPrefix(hdr.Name, ".SIGN") {
			ret.sig, err = handleSig(tarReader)
			if err != nil {
				return apkIndex{}, err
			}
			continue
		}

		// Handle the DESCRIPTION file
		if hdr.Name == "DESCRIPTION" {
			ret.desc, err = handleDesc(tarReader)
			if err != nil {
				return apkIndex{}, err
			}
			continue
		}

		// Handle the APKINDEX file
		if hdr.Name == "APKINDEX" {
			ret.index, err = handleAPKIndex(tarReader)
			if err != nil {
				return apkIndex{}, err
			}
			continue
		}
	}

	return ret, nil
}

func handlePKGDef(orig string) apkPackage {
	lines := strings.Split(orig, "\n")
	var ret apkPackage
	for _, line := range lines {
		// split left and right
		lr := strings.SplitN(line, ":", 2)
		if len(lr) < 2 {
			continue
		}

		left, right := lr[0], lr[1]
		if left == "P" {
			ret.name = right
			continue
		}

		if left == "V" {
			arr := apkVerRegex.FindAllStringSubmatch(right, 1)
			ret.version = arr[0][1]
			ret.revision = arr[0][2]
			continue
		}

		if left == "m" {
			arr := maintainerRegex.FindAllStringSubmatch(right, 1)
			ret.maintainerName = arr[0][1]
			ret.maintainerEmail = arr[0][2]
			continue
		}
	}
	return ret
}

func handleAPKIndex(r io.Reader) (map[string]apkPackage, error) {
	var buf bytes.Buffer
	_, err := io.Copy(&buf, r)
	if err != nil {
		return nil, err
	}

	pkgDefStrings := strings.Split(buf.String(), "\n\n")
	pkgDefs := make(map[string]apkPackage)
	for _, def := range pkgDefStrings {
		pkg := handlePKGDef(def)
		pkgDefs[pkg.name] = pkg
	}

	return pkgDefs, nil
}

func handleDesc(r io.Reader) (string, error) {
	var buf bytes.Buffer
	_, err := io.Copy(&buf, r)
	if err != nil {
		return "", err
	}

	return buf.String(), nil
}

func handleSig(r io.Reader) ([]byte, error) {
	var buf bytes.Buffer
	_, err := io.Copy(&buf, r)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func filterForMaintainer(maintainerEmail string, idxs ...apkIndex) map[string]apkPackage {
	pkgs := make(map[string]apkPackage)
	for _, idx := range idxs {
		for name, pkg := range idx.index {
			if pkg.maintainerEmail == maintainerEmail {
				pkgs[name] = pkg
			}
		}
	}
	return pkgs
}
