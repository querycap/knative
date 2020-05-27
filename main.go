package main

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"sort"
	"strings"

	"gopkg.in/yaml.v2"
)

func main() {
	c, err := loadComponents()
	if err != nil {
		panic(err)
	}

	write("operator", map[string]string{
		"operator": c.Operator,
	})
	write("serving", c.Serving)
	write("eventing", c.Eventing)
}

func write(group string, components map[string]string) {
	workflow := NewWorkflow(group)
	componentNames := make([]string, 0)

	for componentName := range components {
		componentNames = append(componentNames, componentName)
	}

	sort.Strings(componentNames)

	for _, componentName := range componentNames {
		writeDockerfile(group, componentName, components[componentName])
		writeJobToWorkflow(workflow, group, componentName)
	}

	ioutil.WriteFile(fmt.Sprintf("./.github/workflows/build-%s.yml", group), workflow.Bytes(), os.ModePerm)
}

func writeDockerfile(group string, componentName string, pkg string) {
	repo := "https://github.com/knative/" + strings.Split(pkg, "/")[1] + ".git"
	pkgRootName := strings.Join(strings.Split(pkg, "/")[0:2], "/")

	hasKoLocalData := false

	fileInfo, err := os.Stat(path.Join(group, componentName, "kodata"))
	if err == nil {
		hasKoLocalData = fileInfo.IsDir()
	}

	buf := bytes.NewBufferString(`
FROM golang:1.14-alpine as build-env

RUN apk add --no-cache git

ARG GOPROXY=https://proxy.golang.org,direct
ARG VERSION=v0.0.1

WORKDIR /go/src
RUN git clone ` + repo + ` ` + pkgRootName + `; 

WORKDIR /go/src/` + pkg + `

RUN set -eux; \
    \
	git checkout ${VERSION}; \
    go build -o ` + componentName + `; \
    \
    mkdir -p /app; \
    mkdir -p ./kodata; \
    cp -R ./kodata /app/kodata; \
    cp ` + componentName + ` /app/;

FROM alpine
COPY --from=build-env /app /app
`)

	if hasKoLocalData {
		_, _ = io.WriteString(buf, `COPY ./`+path.Join(componentName, "kodata")+` /app/kodata
`)

	}

	_, _ = io.WriteString(buf, `ENV KO_DATA_PATH=/app/kodata
CMD ["/app/`+componentName+`"]
`)

	ioutil.WriteFile(fmt.Sprintf("./%s/Dockerfile.%s", group, componentName), buf.Bytes(), os.ModePerm)
}

func writeJobToWorkflow(writer io.Writer, group string, componentName string) {
	tagName := "querycap/knative-" + group + `-` + componentName
	if group == componentName {
		tagName = "querycap/knative-" + componentName
	}

	_, _ = fmt.Fprintf(writer, `
  build-`+componentName+`:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2
      - uses: crazy-max/ghaction-docker-buildx@v3
        with:
          buildx-version: v0.4.1
      - run: echo ${{ secrets.DOCKER_PASSWORD }} | docker login -u ${{ secrets.DOCKER_USERNAME }} --password-stdin
      - run: |
          export VERSION=$(cat ./`+group+`/.version);
          docker buildx build \
                --push \
                --build-arg=VERSION=${VERSION} \
                --platform=linux/amd64,linux/arm64 \
                -t `+tagName+`:${VERSION} \
                -f `+group+`/Dockerfile.`+componentName+` .
      - run: rm -rf ${HOME}/.docker/config
`)
}

func NewWorkflow(group string) *bytes.Buffer {
	return bytes.NewBufferString(`name: build-` + group + `
on:
  push:
    paths:
      - .github/workflows/build-` + group + `.yml
      - ` + group + `/**

jobs:
`)
}

type Components struct {
	Operator string            `yaml:"operator"`
	Serving  map[string]string `yaml:"serving"`
	Eventing map[string]string `yaml:"eventing"`
}

func loadComponents() (*Components, error) {
	file, err := os.Open("./components.yaml")
	if err != nil {
		return nil, err
	}
	defer file.Close()

	c := &Components{}
	return c, yaml.NewDecoder(file).Decode(c)
}
