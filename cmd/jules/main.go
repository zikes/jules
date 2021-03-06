//  This file is part of "jules".
//
//  "jules" is free software: you can redistribute it and/or modify
//  it under the terms of the GNU General Public License as published by
//  the Free Software Foundation, either version 3 of the License, or
//  (at your option) any later version.
//
//  "jules" is distributed in the hope that it will be useful,
//  but WITHOUT ANY WARRANTY; without even the implied warranty of
//  MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
//  GNU General Public License for more details.
//
//  You should have received a copy of the GNU General Public License
//  along with "jules".  If not, see <http://www.gnu.org/licenses/>.

package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"github.com/zikes/multistatus"
	"log"
	"os"
	"strings"
	"sync"
)

func init() {
	log.SetOutput(os.Stdout)
}

func run(stage string, projects []string, conf *Config, args *Arguments) error {
	workerSet := multistatus.New()
	errors := map[string]error{}
	mutex := &sync.Mutex{}
	for _, p := range projects {
		worker := workerSet.Add(fmt.Sprintf("Project: %s", p))
		go func(w *multistatus.Worker, project string) {
			cmd, err := GetCommand(stage, project, conf)

			if err != nil {
				mutex.Lock()
				errors[project] = err
				mutex.Unlock()
				w.Fail()
				return
			}

			var buff bytes.Buffer
			err = ExecuteCommand(stage, project, &buff, cmd)

			if err != nil {
				mutex.Lock()
				errors[project] = err
				mutex.Unlock()
				w.Fail()
				log.Println(buff.String())
				return
			}

			w.Done()
		}(worker, p)
	}
	// Print the WorkerSet's status until all Workers have completed
	workerSet.Print(context.Background())
	mutex.Lock()
	if len(errors) != 0 {
		for project, err := range errors {
			log.Printf("Error with project %s:\n%s", project, err.Error())
		}
		os.Exit(1)
	}
	mutex.Unlock()
	return nil
}

func main() {
	args := GetArguments()

	// * Assemble a list of map[string]map[string] based on config.
	conf, err := ReadConfig(args.ConfigPath)

	if err != nil {
		log.Fatal(err.Error())
	}

	// If the user did not specify a stage, then show the usage.
	if args.Stage == "" {
		flag.Usage()
		return
	}

	// Lint?
	for _, v := range os.Args {
		if strings.ToLower(v) == "lint" {
			lint(conf)
			return
		}
		if strings.ToLower(v) == "help" {
			help()
			return
		}
	}

	// * Create an array of projects to be run based on the arguments
	var (
		projects []string
		stage    string
	)

	projects = args.Projects
	stage = args.Stage

	// If no projects were specified, then use all of them
	if len(args.Projects) == 0 {
		i := 0
		projects = make([]string, len(conf.Projects))
		for k, _ := range conf.Projects {
			projects[i] = k
			i++
		}
	}

	err = run(stage, projects, conf, args)
	if err != nil {
		log.Fatal(err.Error())
	}
	return
}
