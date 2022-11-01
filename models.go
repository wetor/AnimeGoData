package main


type Subject struct {
	ID int64 `json:"id"`
	Type int `json:"type"`
	Name string `json:"name"`
	NameCN string `json:"name_cn"`
	Platform int `json:"platform"`

	Eps int `json:"eps"`
	AirDate string `json:"airdate"`
}

type Episode struct {
	ID int64 `json:"id"`
	SubjectID int64 `json:"subject_id"`
	AirDate string `json:"airdate"`
	Sort float32 `json:"sort"`
	Type int `json:"type"`
}
