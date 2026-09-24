package main

func main() {
	defer func() {
		v := recover()
		e := v.(interface {
			Error() string
			RuntimeError()
		})
		e.RuntimeError()
		println(e.Error(), v == v)
	}()
	var v any = 3
	_ = v.(string)
}
