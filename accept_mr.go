package main

import (
	"fmt"
	"net/http"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/xanzy/go-gitlab"
)

type AcceptMr struct {
	Client             *gitlab.Client
	OnBuildSucceed     bool
	RemoveSourceBranch bool
	PipelineName       string
	PipelineState      string
	ProjectName        string
	FailOnError        bool
	Message            string
}

func (a AcceptMr) Run() error {
	options := &gitlab.AcceptMergeRequestOptions{}
	if a.RemoveSourceBranch {
		options.ShouldRemoveSourceBranch = &a.RemoveSourceBranch
	}
	state := "opened"
	mrs, _, err := a.Client.MergeRequests.ListProjectMergeRequests(a.ProjectName, &gitlab.ListProjectMergeRequestsOptions{
		State: &state,
	})
	if err != nil {
		return err
	}
	log.Infof("On build succeed: %t", a.OnBuildSucceed)
	log.Infof("Remove source branch: %t", a.RemoveSourceBranch)
	nbErrors := 0
	for _, mr := range mrs {
		entry := log.WithFields(log.Fields(map[string]interface{}{
			"title": mr.Title,
		}))
		if mr.WorkInProgress {
			entry.Warn("Skipping merge request, it is in WIP")
			continue
		}
		if a.OnBuildSucceed && mr.MergeWhenPipelineSucceeds {
			continue
		}
		entry.Info("Accepting merge request ...")
		err := a.accept(mr, options)
		if err != nil {
			nbErrors++
			entry.Error(err.Error())
		}
		entry.Info("Finished accepting merge request ...")
	}
	if a.FailOnError && nbErrors > 0 {
		return fmt.Errorf("You have %d merge request which can't be accepted", nbErrors)
	}
	return nil
}

func (a AcceptMr) accept(mr *gitlab.MergeRequest, opt *gitlab.AcceptMergeRequestOptions) error {
	if a.OnBuildSucceed {
		return a.acceptBuildSucceed(mr, opt)
	}
	return a.acceptMrRequest(mr, opt)
}

func (a AcceptMr) acceptMrRequest(mr *gitlab.MergeRequest, opt *gitlab.AcceptMergeRequestOptions) error {
	info, resp, err := a.Client.MergeRequests.AcceptMergeRequest(a.ProjectName, mr.IID, opt)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusMethodNotAllowed {
			return fmt.Errorf("Merging process is blocked (grey button on MR web view). MR is probably in unresolved thread state.")
		}
		return fmt.Errorf("Error occurred while accepting: %s ", err.Error())
	}

	if a.Message != "" {
		_, _, err := a.Client.Notes.CreateMergeRequestNote(a.ProjectName, mr.IID, &gitlab.CreateMergeRequestNoteOptions{
			Body: &a.Message,
		})
		if err != nil {
			return fmt.Errorf("Error when commenting on merge request: %s ", err.Error())
		}
	}

	if len(info.MergeError) != 0 {
		log.WithFields(log.Fields(map[string]interface{}{
			"title": mr.Title,
		})).Warnf("could not merge request due to merge error: %s", info.MergeError)

		// Best effort, no error checking
		newTitle := "WIP: " + mr.Title
		a.Client.MergeRequests.UpdateMergeRequest(a.ProjectName, mr.IID, &gitlab.UpdateMergeRequestOptions{
			Title: &newTitle,
		})

		// Best effort, no error checking
		msg := "Could not merge automatically due to merge error: " + info.MergeError
		a.Client.Notes.CreateMergeRequestNote(a.ProjectName, mr.IID, &gitlab.CreateMergeRequestNoteOptions{
			Body: &msg,
		})
	}
	return nil
}

func (a AcceptMr) acceptBuildSucceed(mr *gitlab.MergeRequest, opt *gitlab.AcceptMergeRequestOptions) error {
	statuses, _, _ := a.Client.Commits.GetCommitStatuses(a.ProjectName, mr.SHA, nil)
	if statuses != nil && len(statuses) > 0 && statuses[0].Status == string(gitlab.Success) {
		return a.acceptMrRequest(mr, opt)
	}
	err := a.updateCommitStatus(statuses, mr.SHA)
	if err != nil {
		return fmt.Errorf("Error occurred while changing status: %s ", err.Error())
	}
	return nil
}

func (a AcceptMr) updateCommitStatus(statuses []*gitlab.CommitStatus, sha string) error {
	if statuses != nil && len(statuses) > 0 && statuses[0].Status != "" {
		return nil
	}
	if a.PipelineName == "" && a.PipelineState == "" {
		return nil
	}
	if a.PipelineState == "" {
		a.PipelineState = "running"
	}
	if a.PipelineName == "" {
		a.PipelineState = "accept-mr"
	}

	state := strings.ToLower(a.PipelineState)
	stateValue := gitlab.BuildStateValue(state)
	_, _, err := a.Client.Commits.SetCommitStatus(a.ProjectName, sha, &gitlab.SetCommitStatusOptions{
		State: *gitlab.BuildState(stateValue),
		Name:  &a.PipelineName,
	})
	if err != nil {
		return err
	}
	return nil
}
