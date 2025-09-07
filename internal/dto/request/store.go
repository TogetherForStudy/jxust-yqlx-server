package request

type GetFileURLRequest struct {
	Download bool `form:"download"`
	Expires  int  `form:"expires"`
}
