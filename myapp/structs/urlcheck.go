package structs

import (
	"net/http"
	"time"
)

type URLCheck struct {
	URL    string
	Status string
	Code   int
}

type PageVisit struct {
	PageURL   string
	Requests  []*APIRequest
	VisitTime time.Time
}

type APIRequest struct {
	URL         string
	Method      string
	StatusCode  int
	ReqHeaders  map[string]string
	RespHeaders map[string]string
	Body        string
	Timestamp   time.Time
}

type SnapshotAction struct {
	Type      string `json:"type"`                
	Selector  string `json:"selector,omitempty"`  
	Value     string `json:"value,omitempty"`     
	Timestamp int64  `json:"timestamp,omitempty"` 
	URL       string `json:"url,omitempty"`      
}

type Snapshot struct {
	ID        string           `json:"id"`
	URL       string           `json:"url"`
	Name      string           `json:"name,omitempty"`
	Actions   []SnapshotAction `json:"actions"`
	CreatedAt time.Time        `json:"created_at"`
}


func (u *URLCheck) Check() {
	resp, err := http.Get(u.URL)
	if err != nil || resp.StatusCode != 200 {
		u.Status = "DOWN"
		u.Code = 400
	} else {
		u.Status = "UP"
		u.Code = 200
	}
}

func NewPageVisit(url string) *PageVisit {
	return &PageVisit{
		PageURL:   url,
		Requests:  []*APIRequest{},
		VisitTime: time.Now(),
	}
}

func NewAPIRequest(url, method string, status int, reqHeaders, respHeaders map[string]string, body string) *APIRequest {
	return &APIRequest{
		URL:         url,
		Method:      method,
		StatusCode:  status,
		ReqHeaders:  reqHeaders,
		RespHeaders: respHeaders,
		Body:        body,
		Timestamp:   time.Now(),
	}
}

