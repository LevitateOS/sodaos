package api

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/levitateos/factory/pkg/factory/control"
	"github.com/levitateos/sodaos/internal/web/auth"
)

// Called only after the existing raw-body HMAC check. Notifications can wake
// an admitted continuation, never supply an execution or spending capability.
func (h IntakeHandler) pullEvent(w http.ResponseWriter,r *http.Request,event,delivery string,body []byte) bool {
	switch event {
	case "pull_request","pull_request_sync","pull_request_comment","pull_request_review","pull_request_review_approved","pull_request_review_rejected","pull_request_review_comment","pull_request_review_request","issue_comment","status","check_run","check_suite":
	default: return false
	}
	var payload struct {
		Repository *intakeRepository `json:"repository"`
		Number int64 `json:"number"`
		Issue *intakeIssue `json:"issue"`
		IsPull bool `json:"is_pull"`
		SHA string `json:"sha"`
		Pull *struct {
			Number int64 `json:"number"`
			Head *struct { SHA string `json:"sha"` } `json:"head"`
		} `json:"pull_request"`
		CheckRun *struct { Head string `json:"head_sha"` } `json:"check_run"`
		CheckSuite *struct { Head string `json:"head_sha"` } `json:"check_suite"`
	}
	if json.Unmarshal(body,&payload)!=nil { auth.JSONError(w,400,"invalid_hint","Invalid PR event locator."); return true }
	if event=="issue_comment" && !payload.IsPull { return false }
	number,head:=payload.Number,payload.SHA
	if payload.Pull!=nil {
		if number>0 && payload.Pull.Number>0 && number!=payload.Pull.Number { auth.JSONError(w,400,"invalid_hint","Conflicting PR event locators."); return true }
		if payload.Pull.Number>0 { number=payload.Pull.Number }
		if payload.Pull.Head!=nil { head=payload.Pull.Head.SHA }
	}
	if number==0 && payload.Issue!=nil { number=payload.Issue.Index }
	if event=="check_run" && payload.CheckRun!=nil { head=payload.CheckRun.Head }
	if event=="check_suite" && payload.CheckSuite!=nil { head=payload.CheckSuite.Head }
	if payload.Repository==nil || payload.Repository.ID<1 { auth.JSONError(w,400,"invalid_hint","Native repository identity is required."); return true }
	hint:=control.PullEventHint{Delivery:delivery,Repository:payload.Repository.ID,Number:number,Head:head}
	if hint.Validate()!=nil { auth.JSONError(w,400,"invalid_hint","PR number or check commit is required."); return true }
	if h.Identities==nil { auth.JSONError(w,503,"intake_unavailable","Factory identity mapping is unavailable."); return true }
	repository,err:=h.Identities.LocalRepository(r.Context(),hint.Repository)
	if err!=nil { auth.JSONError(w,503,"intake_unavailable","Factory repository identity is unavailable."); return true }
	hint.Repository=int64(repository)
	owner,ok:=h.Coordinator.(interface { ObservePullEvent(context.Context,control.PullEventHint) (bool,error) })
	if !ok { auth.JSONError(w,503,"intake_unavailable","PR wake reconciliation is unavailable."); return true }
	queued,err:=owner.ObservePullEvent(r.Context(),hint)
	if err!=nil { auth.JSONError(w,503,"intake_unavailable","PR wake reconciliation is pending."); return true }
	auth.JSONResponse(w,200,struct { Queued bool `json:"queued"`; Ignored bool `json:"ignored"` }{queued,!queued})
	return true
}
