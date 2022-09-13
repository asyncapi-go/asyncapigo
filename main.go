package main

import (
	_ "embed"
	"flag"
	"fmt"
	"log"
	"os"

	"gopkg.in/yaml.v3"

	"github.com/asyncapi-go/asyncapigo/model"
	"github.com/asyncapi-go/asyncapigo/parser"
)

func main() {
	var api model.AsyncApi
	api.AsyncApi = "2.4.0"
	api.Channels = make(map[string]model.Channel)
	api.Components.Messages = make(map[string]model.Message)
	api.Components.Schemas = make(map[string]model.Object)

	var path string
	var out string
	flag.StringVar(&path, "d", "", "dir")
	flag.StringVar(&out, "out", "docs/asyncapi.yaml", "dir")
	flag.Parse()

	workDir, err := os.Getwd()
	if err != nil {
		log.Fatal("cannot get working directory: ", err)
	}

	parser := parser.New(workDir, &api)
	fmt.Println("Generating api...")
	err = parser.Parse(path, 2)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Preparing output...")
	bytes, err := yaml.Marshal(api)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Saving...")
	outFile, err := os.Create(out)
	if err != nil {
		log.Fatalf("create file: %v", err)
	}
	defer outFile.Close()

	_, err = outFile.Write(bytes)
	if err != nil {
		log.Fatal(err)
	}
}
