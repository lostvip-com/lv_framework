package lv_dto

type Page struct {
	List  interface{} `json:"list"`
	Count int         `json:"count"`
	PageReq
}
