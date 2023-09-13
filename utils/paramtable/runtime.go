package paramtable

var params ComponentParam

func Init() {
	params.Init()
}

func Get() *ComponentParam {
	return &params
}
