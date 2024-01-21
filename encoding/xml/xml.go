package xml

import (
	"encoding/xml"
	"fmt"
	"strings"

	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/kun/exception"
)

// ProcessEncode xml Encode
func ProcessEncode(process *process.Process) interface{} {
	process.ValidateArgNums(1)
	var data = process.ArgsMap(0)
	res, err := mapToXML(data)
	if err != nil {
		exception.New("XML decode error: %s", 500, err).Throw()
	}
	return fmt.Sprintf("<xml>\n%s\n</xml>", res)
}

// ProcessDecode xml Decode
func ProcessDecode(process *process.Process) interface{} {
	process.ValidateArgNums(1)
	res, err := xmlToMap(process.ArgsString(0))
	if err != nil {
		exception.New("XML decode error: %s", 500, err).Throw()
	}
	return res
}

func mapToXML(data map[string]interface{}) ([]byte, error) {
	type entry struct {
		XMLName xml.Name
		Value   interface{} `xml:",chardata"`
	}

	entries := make([]entry, 0, len(data))
	for key, value := range data {
		entries = append(entries, entry{XMLName: xml.Name{Local: key}, Value: value})
	}

	xmlData, err := xml.MarshalIndent(entries, "  ", "  ")
	if err != nil {
		return nil, err
	}

	return xmlData, nil
}

func xmlToMap(data string) (map[string]interface{}, error) {
	decoder := NewDecoder(strings.NewReader(data))
	result, err := decoder.Decode()
	if err != nil {
		return nil, err
	}
	return result, nil
}
