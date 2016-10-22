package raopd

type logger interface {
	Println(d ...interface{})
}

func Debug(name string, value interface{}) {
	switch name {
	case "sequencelog":
		flag, _ := value.(bool)
		debugSequenceLogFlag = flag
	}

}
