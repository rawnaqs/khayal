package version

var (
	Version   = "0.1.0"
	Commit    = ""
	BuildDate = ""
)

func Get() string {
	return Version
}
