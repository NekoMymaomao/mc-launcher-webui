package core


type Request struct {
	Cmd  string         `json:"cmd"`
	Args map[string]any `json:"args,omitempty"`
	Cb   string         `json:"cb,omitempty"`
}

type Response struct {
	Ok    bool     `json:"ok"`
	Data  any      `json:"data,omitempty"`
	Error string   `json:"error,omitempty"`
}

type Progress struct {
	ID    string `json:"id"`
	Done  int64  `json:"done"`
	Total int64  `json:"total"`
	Msg   string `json:"msg"`
}

type Settings struct {
	MaxMemory        int      `json:"maxMemory"`
	MinMemory        int      `json:"minMemory"`
	JavaPath         string   `json:"javaPath"`
	JVMArgs          []string `json:"jvmArgs"`
	GameArgs         []string `json:"gameArgs"`
	DownloadSource   string   `json:"source"`
	Width            int      `json:"width"`
	Height           int      `json:"height"`
	VersionIsolation bool     `json:"versionIsolation"`
	Threads          int      `json:"threads"`
	ListenAddr       string   `json:"listenAddr"`
	Lang             string   `json:"lang"`
}

type Account struct {
	Type        string `json:"type"`
	Name        string `json:"name"`
	UUID        string `json:"uuid"`
	AccessToken string `json:"accessToken"`
	Server      string `json:"server,omitempty"`
}

type AccountStore struct {
	Selected string    `json:"selected"`
	Accounts []Account `json:"accounts"`
}

func (a *AccountStore) current() *Account {
	if len(a.Accounts) == 0 {
		return nil
	}
	if a.Selected == "" {
		a.Selected = a.Accounts[0].UUID
	}
	for i := range a.Accounts {
		if a.Accounts[i].UUID == a.Selected {
			return &a.Accounts[i]
		}
	}
	return &a.Accounts[0]
}

func (a *AccountStore) upsert(acc Account) {
	for i := range a.Accounts {
		if a.Accounts[i].UUID == acc.UUID {
			a.Accounts[i] = acc
			a.Selected = acc.UUID
			saveAccounts()
			return
		}
	}
	a.Accounts = append(a.Accounts, acc)
	a.Selected = acc.UUID
	saveAccounts()
}

func (a *AccountStore) remove(uuid string) bool {
	for i := range a.Accounts {
		if a.Accounts[i].UUID == uuid {
			a.Accounts = append(a.Accounts[:i], a.Accounts[i+1:]...)
			if a.Selected == uuid {
				a.Selected = ""
				if len(a.Accounts) > 0 {
					a.Selected = a.Accounts[0].UUID
				}
			}
			saveAccounts()
			return true
		}
	}
	return false
}

type Artifact struct {
	Path string `json:"path,omitempty"`
	URL  string `json:"url,omitempty"`
	SHA1 string `json:"sha1,omitempty"`
	Size int64  `json:"size,omitempty"`
}

type OSRule struct {
	Name    string `json:"name,omitempty"`
	Arch    string `json:"arch,omitempty"`
	Version string `json:"version,omitempty"`
}

type Rule struct {
	Action   string            `json:"action"`
	OS       *OSRule           `json:"os,omitempty"`
	Features map[string]bool   `json:"features,omitempty"`
}

type Downloads struct {
	Artifact    Artifact              `json:"artifact,omitempty"`
	Classifiers map[string]Artifact   `json:"classifiers,omitempty"`
}

type Extract struct {
	Exclude []string `json:"exclude,omitempty"`
}

type Library struct {
	Name         string            `json:"name"`
	Downloads    *Downloads        `json:"downloads,omitempty"`
	Natives      map[string]string `json:"natives,omitempty"`
	Rules        []Rule            `json:"rules,omitempty"`
	Extract      *Extract          `json:"extract,omitempty"`
	ShouldUse    bool              `json:"-"`
	IsNative     bool              `json:"-"`
	MavenBase    string            `json:"-"`
	ResolvedPath string            `json:"-"`
	ResolvedURL  string            `json:"-"`
	NativePath   string            `json:"-"`
	NativeURL    string            `json:"-"`
}

type VersionDetail struct {
	Valid        bool      `json:"valid"`
	ID           string    `json:"id"`
	Type         string    `json:"type"`
	MainClass    string    `json:"mainClass"`
	Assets       string    `json:"assets"`
	AssetIndexID string    `json:"assetIndex"`
	AssetIndex   Artifact  `json:"-"`
	Client       Artifact  `json:"-"`
	Libraries    []Library `json:"-"`
	JVMArgs      []string  `json:"-"`
	GameArgs     []string  `json:"-"`
	LegacyArgs   string    `json:"minecraftArguments"`
	JavaMajor    int       `json:"javaVersion"`
	LogArgument  string    `json:"loggingClientArgument"`
	LogFile      Artifact  `json:"-"`
	RawJSON      string    `json:"-"`
}

type VersionEntry struct {
	ID   string `json:"id"`
	Type string `json:"type"`
	URL  string `json:"url"`
	Time string `json:"time"`
}

type VersionManifest struct {
	Latest   struct{ Release, Snapshot string } `json:"latest"`
	Versions []VersionEntry                   `json:"versions"`
}
