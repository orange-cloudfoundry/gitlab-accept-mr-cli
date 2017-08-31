package main

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/xanzy/go-gitlab"
	"strings"
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
	mrs, _, err := a.Client.MergeRequests.ListMergeRequests(a.ProjectName, &gitlab.ListMergeRequestsOptions{
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
	if a.Message != "" {
		_, _, err := a.Client.Notes.CreateMergeRequestNote(a.ProjectName, mr.IID, &gitlab.CreateMergeRequestNoteOptions{
			Body: &a.Message,
		})
		if err != nil {
			return fmt.Errorf("Error when commenting on merge request: %s ", err.Error())
		}
	}
	_, _, err := a.Client.MergeRequests.AcceptMergeRequest(a.ProjectName, mr.IID, opt)
	if err != nil {
		return fmt.Errorf("Error occurred while accepting: %s ", err.Error())
	}
	return nil
}
func (a AcceptMr) acceptBuildSucceed(mr *gitlab.MergeRequest, opt *gitlab.AcceptMergeRequestOptions) error {
	statuses, _, _ := a.Client.Commits.GetCommitStatuses(a.ProjectName, mr.Sha, nil)
	if statuses != nil && len(statuses) > 0 && statuses[0].Status == string(gitlab.Success) {
		return a.acceptMrRequest(mr, opt)
	}
	err := a.updateCommitStatus(statuses, mr.Sha)
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
	_, _, err := a.Client.Commits.SetCommitStatus(a.ProjectName, sha, &gitlab.SetCommitStatusOptions{
		State: gitlab.BuildState(strings.ToLower(a.PipelineState)),
		Name:  &a.PipelineName,
	})
	if err != nil {
		return err
	}
	return nil
}
