package envload

type Loader struct {
	original []iterItem
	envdir   string
}

type Iterator struct {
	ch    chan *iterItem
	nextK string
	nextV string
}

type iterItem struct {
	key   string
	value string
}

type Environment interface {
	Clearenv()
	Setenv(string, string)
}
type sysenv struct{}

type Option interface {
	Name() string
	Value() interface{}
}

const (
	ContextKey = "ContextKey"
	EnvironmentKey = "EnvironmentKey"
	LoadEnvdirKey = "LoadEnvdirKey"
)

type option struct {
	name  string
	value interface{}
}
