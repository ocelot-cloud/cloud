package backups

type BackupCreationDto struct {
	Maintainer               string
	AppName                  string
	VersionName              string
	VersionCreationTimestamp string
	Description              string
	VersionZipContent        []byte
}

type DockerComposeYaml struct {
	Volumes map[string]interface{} `yaml:"volumes"`
}

type Snapshot struct {
	Time string   `json:"time"`
	Tags []string `json:"tags"`
	Id   string   `json:"id"`
}
