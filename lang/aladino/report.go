// Copyright 2022 Explore.dev Unipessoal Lda. All Rights Reserved.
// Use of this source code is governed by a license that can be
// found in the LICENSE file.

package aladino

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/google/go-github/v45/github"
	"github.com/reviewpad/reviewpad/v3/engine"
	"github.com/reviewpad/reviewpad/v3/utils"
	"github.com/reviewpad/reviewpad/v3/utils/fmtio"
)

type Report struct {
	Actions []string
}

const ReviewpadReportCommentAnnotation = "<!--@annotation-reviewpad-report-->"

func reportError(format string, a ...interface{}) error {
	return fmtio.Errorf("report", format, a...)
}

func (report *Report) addToReport(statement *engine.Statement) {
	report.Actions = append(report.Actions, statement.GetStatementCode())
}

func ReportHeader(safeMode bool) string {
	var sb strings.Builder

	// Annotation
	sb.WriteString(fmt.Sprintf("%v\n", ReviewpadReportCommentAnnotation))
	// Header
	if safeMode {
		sb.WriteString("**Reviewpad Report** (Reviewpad ran in dry-run mode because configuration has changed)\n\n")
	} else {
		sb.WriteString("**Reviewpad Report**\n\n")
	}

	return sb.String()
}

func buildReport(safeMode bool, report *Report) string {
	var sb strings.Builder

	sb.WriteString(ReportHeader(safeMode))
	sb.WriteString(BuildVerboseReport(report))

	return sb.String()
}

func BuildVerboseReport(report *Report) string {
	if report == nil {
		return ""
	}

	var sb strings.Builder

	sb.WriteString(":scroll: **Executed actions**\n")

	sb.WriteString("```yaml\n")

	// Report
	for _, action := range report.Actions {
		sb.WriteString(fmt.Sprintf("%v\n", action))
	}

	sb.WriteString("```\n")
	return sb.String()
}

func DeleteReportComment(env Env, commentId int64) error {
	pullRequest := env.GetPullRequest()
	owner := utils.GetPullRequestBaseOwnerName(pullRequest)
	repo := utils.GetPullRequestBaseRepoName(pullRequest)

	_, err := env.GetClient().Issues.DeleteComment(env.GetCtx(), owner, repo, commentId)

	if err != nil {
		return reportError("error on deleting report comment %v", err.(*github.ErrorResponse).Message)
	}

	return nil
}

func UpdateReportComment(env Env, commentId int64, report string) error {
	gitHubComment := github.IssueComment{
		Body: &report,
	}

	pullRequest := env.GetPullRequest()
	owner := utils.GetPullRequestBaseOwnerName(pullRequest)
	repo := utils.GetPullRequestBaseRepoName(pullRequest)

	_, _, err := env.GetClient().Issues.EditComment(env.GetCtx(), owner, repo, commentId, &gitHubComment)

	if err != nil {
		return reportError("error on updating report comment %v", err.(*github.ErrorResponse).Message)
	}

	return nil
}

func AddReportComment(env Env, report string) error {
	gitHubComment := github.IssueComment{
		Body: &report,
	}

	pullRequest := env.GetPullRequest()
	owner := utils.GetPullRequestBaseOwnerName(pullRequest)
	repo := utils.GetPullRequestBaseRepoName(pullRequest)
	prNum := utils.GetPullRequestNumber(pullRequest)

	_, _, err := env.GetClient().Issues.CreateComment(env.GetCtx(), owner, repo, prNum, &gitHubComment)

	if err != nil {
		return reportError("error on creating report comment %v", err.(*github.ErrorResponse).Message)
	}

	return nil
}

func FindReportComment(env Env) (*github.IssueComment, error) {
	pullRequest := env.GetPullRequest()
	owner := utils.GetPullRequestBaseOwnerName(pullRequest)
	repo := utils.GetPullRequestBaseRepoName(pullRequest)
	prNum := utils.GetPullRequestNumber(pullRequest)

	comments, err := utils.GetPullRequestComments(env.GetCtx(), env.GetClient(), owner, repo, prNum, &github.IssueListCommentsOptions{
		Sort:      github.String("created"),
		Direction: github.String("asc"),
	})
	if err != nil {
		return nil, reportError("error getting issues %v", err.(*github.ErrorResponse).Message)
	}

	reviewpadCommentAnnotationRegex := regexp.MustCompile(fmt.Sprintf("^%v", ReviewpadReportCommentAnnotation))

	var reviewpadExistingComment *github.IssueComment

	for _, comment := range comments {
		isReviewpadReportComment := reviewpadCommentAnnotationRegex.Match([]byte(*comment.Body))
		if isReviewpadReportComment {
			reviewpadExistingComment = comment
			break
		}
	}

	return reviewpadExistingComment, nil
}
