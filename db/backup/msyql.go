package backup

import (
	"fmt"
	"io"
	"net"
	"os/exec"

	driver "github.com/go-sql-driver/mysql"
)

// MySQLConfig specifies mysql dump configuration, see the defaults.
type MySQLConfig struct {
	// DB Host (e.g. 127.0.0.1)
	Host string
	// DB Port (e.g. 3306)
	Port string
	// DB Name
	DB string
	// DB User
	User string
	// DB Password
	Password string
	// Extra mysqldump options
	// e.g []string{"--extended-insert"}
	Options []string
}

// NewMySQLConfig creates MySQLConfig based dsn data source connection string.
func NewMySQLConfig(dsn string) (*MySQLConfig, error) {
	c, err := driver.ParseDSN(dsn)
	if err != nil {
		return nil, err
	}

	host, port, _ := net.SplitHostPort(c.Addr)

	return &MySQLConfig{
		Host:     host,
		Port:     port,
		DB:       c.DBName,
		User:     c.User,
		Password: c.Passwd,
	}, nil
}

// MysqlDumpCmd is the path to the `mysqldump` executable
var MysqlDumpCmd = "mysqldump"

// MySQLDefaultOptions specifies additional mysqldump flags.
var MySQLDefaultOptions = []string{
	"--protocol=TCP",
	"--single-transaction",
	"--add-drop-database",
	"--hex-blob",
	"--events",
	"--routines",
	"--triggers",
}

// MySQL dumps MySQL using mysqldump executable,
func MySQL(config *MySQLConfig, w io.Writer) error {
	if w == nil {
		panic("Missing config")
	}

	args := mysqlOptions(config)
	cmd := exec.Command(MysqlDumpCmd, args...)
	cmd.Stdout = w

	return cmd.Run()
}

func mysqlOptions(config *MySQLConfig) []string {
	o := MySQLDefaultOptions
	if config.Options != nil {
		o = append(MySQLDefaultOptions, config.Options...)
	}
	o = append(o, fmt.Sprintf(`-h%v`, config.Host))
	o = append(o, fmt.Sprintf(`-P%v`, config.Port))
	o = append(o, fmt.Sprintf(`-u%v`, config.User))
	o = append(o, fmt.Sprintf(`-p%v`, config.Password))
	o = append(o, config.DB)

	return o
}
