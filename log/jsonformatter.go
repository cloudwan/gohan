package log

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/op/go-logging"
)

type jsonFormatter struct {
	componentName string
}

//Format formats message to JSON format
func (f *jsonFormatter) Format(calldepth int, record *logging.Record, output io.Writer) error {
	result := map[string]interface{}{
		"timestamp":      record.Time,
		"log_level":      record.Level.String(),
		"log_type":       "log",
		"msg":            record.Message(),
		"component_name": record.Module,
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return err
	}
	fmt.Fprintf(output, "%s\n", resultJSON)
	return nil
}
