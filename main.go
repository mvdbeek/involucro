package main

import (
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/fsouza/go-dockerclient"
	file "github.com/thriqon/involucro/file"
	wrap "github.com/thriqon/involucro/steps/wrap"
	"os"
	"path/filepath"
)

func main() {

	arguments := parseArguments()

	client, _ := docker.NewClient(arguments["--host"].(string))
	err := client.Ping()
	if err != nil {
		log.Fatal("Docker not reachable")
	}

	log.SetLevel(log.DebugLevel)
	if arguments["--wrap"] != nil {
		conf := wrap.AsImage{
			SourceDir:         arguments["--wrap"].(string),
			TargetDir:         arguments["--at"].(string),
			ParentImage:       arguments["--into-image"].(string),
			NewRepositoryName: arguments["--as"].(string),
		}
		log.WithFields(log.Fields{"conf": conf}).Debug("Starting wrap")

		err := conf.WithDockerClient(client)
		if err != nil {
			log.WithFields(log.Fields{"error": err}).Panic("Failed wrapping")
		}
		return
	}

	relativeWorkDir := arguments["-w"].(string)
	workingDir, _ := filepath.Abs(relativeWorkDir)
	log.WithFields(log.Fields{"workdir": workingDir}).Info("Start")

	ctx := file.InstantiateRuntimeEnv(workingDir)

	if arguments["-s"].(bool) {
		fmt.Println("#!/bin/sh")
	}

	if arguments["-e"] != nil {
		err := ctx.RunString(arguments["-e"].(string))
		if err != nil {
			log.WithFields(log.Fields{"error": err}).Fatal("Failed executing script")
		}
	} else {
		err := ctx.RunFile(arguments["-f"].(string))
		if err != nil {
			log.WithFields(log.Fields{"error": err}).Fatal("Failed executing file")
		}
	}

	for _, element := range (arguments["<task>"]).([]string) {
		steps := ctx.Tasks[element]
		if len(steps) == 0 {
			log.WithFields(log.Fields{"task": element}).Warn("no steps defined for task")
		}
		for _, step := range steps {
			if arguments["-n"].(bool) {
				step.DryRun()
			} else if arguments["-s"].(bool) {
				step.AsShellCommandOn(os.Stdout)
			} else {
				if err := step.WithDockerClient(client); err != nil {
					log.WithFields(log.Fields{"error": err}).Fatal("Error during task processing")
				}
			}
		}
	}
}
