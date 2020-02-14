package checkpoint

type Identity struct {
	ID  int  `json:"id"`
	God bool `json:"god"`
}

type Profile struct {
	Name *string `json:"name"`
}
