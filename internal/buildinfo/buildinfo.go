package buildinfo

const (
	Graffiti       = "             ._____.                   ___.           __   \n  ___________|__\\_ |__   ____          \\_ |__   _____/  |_ \n_/ ___\\_  __ \\  || __ \\_/ __ \\   ______ | __ \\ /  _ \\   __\\\n\\  \\___|  | \\/  || \\_\\ \\  ___/  /_____/ | \\_\\ (  <_> )  |  \n \\___  >__|  |__||___  /\\___  >         |___  /\\____/|__|  \n     \\/              \\/     \\/              \\/             "
	GreetingCLI    = "\nversion: %s \nbuild time: %s\ntg: %s\ngithub: %s\n"
	GithubBloopURL = "https://github.com/robotomize/cribe.git"
	TgBloopURL     = "https://t.me/cribe_bot"
)

var (
	BuildTag = "v0.0.0"
	Name     = "cribebot"
	Time     = ""
)

type buildinfo struct{}

func (buildinfo) Tag() string {
	return BuildTag
}

func (buildinfo) Name() string {
	return Name
}

func (buildinfo) Time() string {
	return Time
}

var Info buildinfo
