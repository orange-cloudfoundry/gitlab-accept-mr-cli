package main

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli"
	"github.com/xanzy/go-gitlab"
)

func main() {
	app := cli.NewApp()
	app.Name = "accept-mr"
	app.Version = "1.0.0"
	app.Usage = "Automatically accept Merge Request on project"
	app.ErrWriter = os.Stderr
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:   "url, u",
			Usage:  "Url to your gitlab",
			EnvVar: "GITLAB_URL",
		},
		cli.StringFlag{
			Name:   "token, t",
			Usage:  "User token to access the api",
			EnvVar: "GITLAB_TOKEN",
		},
		cli.StringFlag{
			Name:   "project, p",
			Usage:  "Project name where accepting mr (e.g.: owner/repo)",
			EnvVar: "GITLAB_PROJECT",
		},
		cli.StringFlag{
			Name:  "pipeline-name, pn",
			Usage: "Set a default pipeline name when using on-build-succeed option",
		},
		cli.StringFlag{
			Name:  "pipeline-state, ps",
			Usage: "Set a default pipeline state when using on-build-succeed option (can be pending or running)",
		},
		cli.StringFlag{
			Name:  "message, m",
			Usage: "Set a merge commit message",
		},
		cli.BoolFlag{
			Name:  "failed-on-error, e",
			Usage: "If set accept in error exit with status code > 0",
		},
		cli.BoolFlag{
			Name:  "insecure, k",
			Usage: "Ignore certificate validation",
		},
		cli.BoolFlag{
			Name:  "log-json, j",
			Usage: "Write log in json",
		},
		cli.BoolFlag{
			Name:  "no-color",
			Usage: "Logger will not display colors",
		},
		cli.BoolFlag{
			Name:  "remove-source-branch, rb",
			Usage: "If set it will remove all the time the source branch when merging",
		},
		cli.BoolFlag{
			Name:  "on-build-succeed, bs",
			Usage: "Merge request will automatically accepted if pipeline succeeded",
		},
	}
	app.Action = acceptMrAction
	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
func checkRequired(c *cli.Context) error {
	if c.GlobalString("url") == "" {
		return fmt.Errorf("Gitlab url can't be empty set with --url or GITLAB_URL env var")
	}
	if c.GlobalString("token") == "" {
		return fmt.Errorf("Gitlab token can't be empty set with --token or GITLAB_TOKEN env var")
	}
	if c.GlobalString("project") == "" {
		return fmt.Errorf("Gitlab project can't be empty set with --project or GITLAB_PROJECT env var")
	}
	return nil
}
func loadClient(c *cli.Context) (*gitlab.Client, error) {

	token := c.GlobalString("token")
	url := c.GlobalString("url")
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: c.GlobalBool("insecure"),
		},
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
			DualStack: true,
		}).DialContext,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		Proxy:                 http.ProxyFromEnvironment,
	}
	if !strings.HasSuffix(url, "/api/v4") {
		url = strings.TrimSuffix(url, "/") + "/api/v4"
	}
	git, err := gitlab.NewClient(token, gitlab.WithBaseURL(url), gitlab.WithHTTPClient(&http.Client{Transport: transport}))
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	return git, err
}

func acceptMrAction(c *cli.Context) error {
	loadLogConfig(c)
	err := checkRequired(c)
	if err != nil {
		return err
	}
	client, err := loadClient(c)
	if err != nil {
		return err
	}
	acceptMr := &AcceptMr{
		Client:             client,
		Message:            c.GlobalString("message"),
		FailOnError:        c.GlobalBool("failed-on-error"),
		OnBuildSucceed:     c.GlobalBool("on-build-succeed"),
		ProjectName:        c.GlobalString("project"),
		PipelineState:      c.GlobalString("pipeline-state"),
		PipelineName:       c.GlobalString("pipeline-name"),
		RemoveSourceBranch: c.GlobalBool("remove-source-branch"),
	}
	return acceptMr.Run()
}

func loadLogConfig(c *cli.Context) {
	if c.GlobalBool("log-json") {
		log.SetFormatter(&log.JSONFormatter{})
	}
	log.SetFormatter(&log.TextFormatter{
		DisableColors: c.GlobalBool("no-color"),
	})

}
