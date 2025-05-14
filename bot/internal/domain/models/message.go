package models

type Message struct {
	Id        string `json:"id"`
	UserId    string `json:"user_id"`
	ChannelId string `json:"channel_id"`
	Message   string `json:"message"`
}
