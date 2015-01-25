// main.go for figdash web tool
// as not API in Fig, built up Fig commands with commend line calls to Docker commands
// could make similar calls to Fig for future commands
//

package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"text/tabwriter"

	"github.com/codegangsta/cli"
	"gopkg.in/yaml.v1"
)

func psCmd(services []*service, c *cli.Context) {
	w := new(tabwriter.Writer)
	w.Init(os.Stdout, 0, 8, 2, '\t', 0)
	fmt.Fprintln(w, "NAME\tCOMMAND\tSTATE\tPORTS")
	for _, s := range services {
		for _, cntr := range s.containers {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t\n",
				cntr.name, cntr.command, cntr.status, cntr.ports)
		}
	}
	w.Flush()
}

func logsCmd(services []*service, c *cli.Context) {
	ch := make(chan string)
	total := 0
	for _, s := range services {
		count, err := s.logs(ch, c.Bool("timestamps"), c.GlobalBool("verbose"))
		if err != nil {
			log.Fatal(err)
		}
		total += count
	}
	if total > 0 {
		for line := range ch {
			fmt.Println(line)
		}
	}
}

func rmCmd(services []*service, c *cli.Context) {
	for _, s := range services {
		s.rmf(c.GlobalBool("verbose"))
	}
}

func startCmd(services []*service, c *cli.Context) {
	for _, s := range services {
		if err := s.start(c.GlobalBool("verbose")); err != nil {
			log.Fatal(err)
		}
	}
}

func stopCmd(services []*service, c *cli.Context) {
	for _, s := range services {
		if err := s.stop(c.GlobalBool("verbose")); err != nil {
			log.Fatal(err)
		}
	}
}

func killCmd(services []*service, c *cli.Context) {
	for _, s := range services {
		if err := s.kill(c.GlobalBool("verbose")); err != nil {
			log.Fatal(err)
		}
	}
}

func parseFile(file string) (map[string]*service, error) {
	buf, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, err
	}
	m := make(map[string]*service)
	if err := yaml.Unmarshal(buf, &m); err != nil {
		return nil, err
	}
	return m, nil
}

func createAction(action func([]*service, *cli.Context)) func(*cli.Context) {
	return func(c *cli.Context) {
		serviceMap, err := parseFile(c.GlobalString("file"))
		if err != nil {
			log.Fatal(err)
		}
		services := []*service{}
		if len(c.Args()) == 0 {
			for name, s := range serviceMap {
				s.init(name, serviceMap)
				services = append(services, s)
			}
		} else {
			for _, name := range c.Args() {
				s, ok := serviceMap[name]
				if !ok {
					log.Fatalf("%s: service does not exist", name)
				}
				s.init(name, serviceMap)
				services = append(services, s)
			}
		}
		psCh := make(chan *psData)
		if err := ps(psCh, c.GlobalBool("verbose")); err != nil {
			log.Fatal(err)
		}
		for psdata := range psCh {
			for _, s := range services {
				if s.matchContainer(psdata.name) {
					cntr, err := newContainerFromPsData(s, psdata)
					if err != nil {
						log.Fatal(err)
					}
					s.containers = append(s.containers, cntr)
					break
				}
			}
		}
		action(services, c)
	}
}

func main() {
	app := cli.NewApp()
	app.Name = "figdash"
	app.Usage = "fig dashboard"
	app.Version = "0.0.1"
        app.Email = "mkobar@rkosecurity.com"
	app.Flags = []cli.Flag{
		cli.BoolFlag{
			Name:  "verbose",
			Usage: "Show more output",
		},
                // version flag support is builtin
		//cli.BoolFlag{
		//	Name:  "version",
		//	Usage: "Print version and exit",
		//},
		cli.StringFlag{
			Name:  "file, f",
			Value: "fig.yml",
			Usage: "Specify an alternate fig file",
                        EnvVar: "FIG_FILE",
		},
		cli.StringFlag{
			Name:  "project-name, p",
			Value: "notset",
			Usage: "Specify an alternate project name",
                        EnvVar: "FIG_PROJECT_NAME",
		},
	}
	app.Commands = []cli.Command{
                // build - NOT supported yet
                // help - NOT supported yet
		{
			Name:   "kill",
			Usage:  "Force stop service containers.",
			Action: createAction(killCmd),
		},
		{
			Name:   "logs",
			Usage:  "View output from services",
			Action: createAction(logsCmd),
			Flags: []cli.Flag{
				cli.BoolFlag{
					Name:  "timestamps, t",
					Usage: "Show timestamps",
				},
			},
		},
                // port - NOT supported
		{
			Name:   "ps",
			Usage:  "List containers",
			Action: createAction(psCmd),
		},
                // pull - NOT supported
		{
			Name:   "rm",
			Usage:  "Remove stopped service containers.",
			Action: createAction(rmCmd),
		},
                // run - NOT supported
                // scale - NOT supported
		{
			Name:   "start",
			Usage:  "Start existing containers for a service.",
			Action: createAction(startCmd),
		},
		{
			Name:   "stop",
			Usage:  "Stop existing containers without removing them.",
			Action: createAction(stopCmd),
		},
                // up - NOT supported - due to logs
	}
	app.Run(os.Args)
}
