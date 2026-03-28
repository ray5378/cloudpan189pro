package configs

type Config struct {
	Port     int          `json:"port,default=12395"`
	DBFile   string       `json:"dbFile,default=data/data.db"`
	LogFile  string       `json:"logFile,default=logs/share.log"`
	MediaDir string       `json:"mediaDir,default=media_dir"`
	DBType   string       `json:"dbType,default=sqlite"`
	MySQL    *MySQLConfig `json:"mysql,omitempty"`
}

type MySQLConfig struct {
	Host   string `json:"host,default=127.0.0.1"`
	Port   int    `json:"port,default=3306"`
	User   string `json:"user,default=root"`
	Pass   string `json:"pass,default=root"`
	DBName string `json:"dbName,default=share"`
}

type RuntimeConfig struct {
	*Config
	BuildDate  string
	Commit     string
	GitBranch  string
	GitSummary string
	Version    string
}
