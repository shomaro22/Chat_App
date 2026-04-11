package controller

type setUserActiveRequest struct {
	Active bool `json:"active"`
}

type setRoomFrozenRequest struct {
	Frozen bool `json:"frozen"`
}
