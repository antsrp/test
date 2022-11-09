package reports

import (
	"fmt"
	"os"
	"time"

	"github.com/pkg/errors"
)

type SummaryCSV struct {
	Name  string
	Value uint64
}

// returns filename
func WriteToCSV(summary []SummaryCSV, path string) (string, error) {
	t := time.Now().Unix()

	name := fmt.Sprintf("%v.csv", t)

	f, err := os.Create(path + "//" + name)
	if err != nil {
		return "", errors.Wrap(err, "can't create csv file")
	}
	defer f.Close()

	for _, s := range summary {
		f.WriteString(fmt.Sprintf("%s;%d\n", s.Name, s.Value))
	}
	return name, nil
}
