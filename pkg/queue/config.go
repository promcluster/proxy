package queue

// Config configuration
type Config struct {
	Type         string `yaml:"type"`
	Name         string `yaml:"name"`
	DataPath     string `yaml:"dataPath"`
	MsgSizeLimit int32  `yaml:"msgSizeLimit"`
}
