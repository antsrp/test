package reports

import "time"

type Operation struct {
	Type    string     `json:"operation_type"`
	Favor   string     `json:"service_name,omitempty"`
	Sum     uint64     `json:"sum"`
	Comment string     `json:"comment"`
	Time    *time.Time `json:"time"`
}
