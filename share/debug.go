package share

var debug = false

func SetDebug(d bool) {
	debug = d
}

func GetDebug() bool {
	return debug
}
