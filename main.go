package main

import (
	"crypto/tls"
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli"
	"github.com/xanzy/go-gitlab"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
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
		cli.BoolFlag{
			Name:  "failed-on-error, e",
			Usage: "If true accept in error exit with status code > 0",
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
			Name:  "on-build-succeed, bs",
			Usage: "Merge request will automatically accepted if pipeline succeeded",
		},
	}
	app.Action = acceptMr
	app.Run(os.Args)
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
		Proxy: http.ProxyFromEnvironment,
	}
	if !strings.HasSuffix(url, "/api/v4") {
		url = strings.TrimSuffix(url, "/") + "/api/v4"
	}
	git := gitlab.NewClient(&http.Client{Transport: transport}, token)
	err := git.SetBaseURL(url)
	return git, err
}
func acceptMr(c *cli.Context) error {
	loadLogConfig(c)
	err := checkRequired(c)
	if err != nil {
		return err
	}
	client, err := loadClient(c)
	if err != nil {
		return err
	}
	projName := c.GlobalString("project")
	state := "opened"
	mrs, _, err := client.MergeRequests.ListMergeRequests(projName, &gitlab.ListMergeRequestsOptions{
		State: &state,
	})
	if err != nil {
		return err
	}
	nbErrors := 0
	buildSucceed := c.GlobalBool("on-build-succeed")
	options := &gitlab.AcceptMergeRequestOptions{}
	if buildSucceed {
		options.MergeWhenPipelineSucceeds = &buildSucceed
	}
	for _, mr := range mrs {
		entry := log.WithFields(log.Fields(map[string]interface{}{
			"title":            mr.Title,
			"on-build-succeed": buildSucceed,
		}))
		if mr.WorkInProgress && !buildSucceed {
			entry.Warn("Skipping merge request, it is in WIP")
			continue
		}
		if buildSucceed && mr.MergeWhenPipelineSucceeds {
			continue
		}
		if !mr.WorkInProgress && buildSucceed {
			_, _, err := client.Commits.SetCommitStatus(projName, mr.Sha, &gitlab.SetCommitStatusOptions{
				State: gitlab.Pending,
			})
			if err != nil {
				entry.Errorf("Error occurred while changing status: %s ", err.Error())
				nbErrors++
				continue
			}
		}
		entry.Info("Accepting merge request ...")
		_, _, err = client.MergeRequests.AcceptMergeRequest(projName, mr.IID, options)
		if err != nil {
			entry.Errorf("Error occurred while accepting: %s ", err.Error())
			nbErrors++
			continue
		}
		entry.Info("Finished accepting merge request ...")
	}
	if c.GlobalBool("failed-on-error") && nbErrors > 0 {
		return fmt.Errorf("You have %d merge request which can't be accepted", nbErrors)
	}
	return nil
}
func loadLogConfig(c *cli.Context) {
	if c.GlobalBool("log-json") {
		log.SetFormatter(&log.JSONFormatter{})
	}
	log.SetFormatter(&log.TextFormatter{
		DisableColors: c.GlobalBool("no-color"),
	})

}
