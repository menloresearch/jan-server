package ptr

func ToString(s string) *string {
	return &s
}

func ToInt(i int) *int {
	return &i
}

func ToBool(b bool) *bool {
	return &b
}
