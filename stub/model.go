package stub

import (
	"bytes"
	"encoding/json"
	"fmt"
	log "github.com/sirupsen/logrus"
	"google.golang.org/protobuf/reflect/protoreflect"
	"reflect"
)

type JsonString string

type EnumType interface {
	Descriptor() protoreflect.EnumDescriptor
}

func isEnum(t reflect.Type) bool {
	inter := reflect.TypeOf((*EnumType)(nil)).Elem()
	return t.Implements(inter)
}

type StubType string

func (j *StubType) UnmarshalJSON(data []byte) error {
	var str *string
	json.Unmarshal(data, &str)
	// defaults to 'mock' when empty to maintain backwards compatibility
	if *str == "" {
		*str = "mock"
	}
	*j = StubType(*str)
	return nil
}

type Stub struct {
	FullMethod string        `json:"fullMethod"`
	Type       StubType      `json:"type"`     // mock | forward - default to mock to maintain backwards compatibility
	Request    *StubRequest  `json:"request"`  // Always required
	Response   *StubResponse `json:"response"` // required if type = mock. Ignored otherwise.
	Forward    *StubForward  `json:"forward"`  // required if type = forward. Ignored otherwise.
}

type StubRequest struct {
	Match    string              `json:"match"` // exact | partial
	Content  JsonString          `json:"content"`
	Metadata map[string][]string `json:"metadata"`
}

func (s StubRequest) String() string {
	data, _ := json.Marshal(s)
	return string(data)
}

type StubResponse struct {
	Type    string         `json:"type"` // success | error
	Content JsonString     `json:"content"`
	Error   *ErrorResponse `json:"error"`
}

type StubForward struct {
	ServerAddress string `json:"serverAddress"`
	Record        bool   `json:"record"`
}

type ErrorResponse struct {
	Code    uint32        `json:"code"`
	Message string        `json:"message"`
	Details *ErrorDetails `json:"details"`
}

type ErrorDetails struct {
	Spec   *ErrorDetailsSpec   `json:"spec"`
	Values []ErrorDetailsValue `json:"values"`
}

type ErrorDetailsValue struct {
	SpecOverride *ErrorDetailsSpec `json:"specOverride"`
	Value        JsonString        `json:"value"`
}

type ErrorDetailsSpec struct {
	Import string `json:"import"`
	Type   string `json:"type"`
}

func (j JsonString) String() string {
	return string(j)
}

func (j *JsonString) UnmarshalJSON(data []byte) error {
	buffer := new(bytes.Buffer)
	err := json.Compact(buffer, data)
	if err != nil {
		log.Errorf("error compacting json: %s", string(data))
	}
	result := JsonString(buffer.String())
	*j = result
	return nil
}

func (j *JsonString) MarshalJSON() ([]byte, error) {
	val := string(*j)
	if val == "" {
		return []byte("{}"), nil
	}
	return []byte(val), nil
}

func (j *JsonString) Matches(other JsonString) bool {
	jsonMap := new(map[string]interface{})
	otherJsonMap := new(map[string]interface{})
	json.Unmarshal([]byte(*j), jsonMap)
	json.Unmarshal([]byte(other), otherJsonMap)
	return jsonStringMatches(*jsonMap, *otherJsonMap, false)
}

func (j *JsonString) Equals(other JsonString) bool {
	jsonMap := new(map[string]interface{})
	otherJsonMap := new(map[string]interface{})
	json.Unmarshal([]byte(*j), jsonMap)
	json.Unmarshal([]byte(other), otherJsonMap)
	return jsonStringMatches(*jsonMap, *otherJsonMap, true)
}

func jsonStringMatches(jsonMap, otherJsonMap map[string]interface{}, mustBeEqual bool) bool {
	if mustBeEqual && len(jsonMap) != len(otherJsonMap) {
		return false
	}
	for key, value := range jsonMap {
		otherValue, found := otherJsonMap[key]
		if !found {
			return false
		}
		valueType := fmt.Sprintf("%T", value)
		otherValueType := fmt.Sprintf("%T", otherValue)
		if valueType != otherValueType {
			return false
		}
		switch valueType {
		case "map[string]interface {}": // object
			if !jsonStringMatches(jsonMap[key].(map[string]interface{}), otherJsonMap[key].(map[string]interface{}), mustBeEqual) {
				return false
			}
			continue
		case "[]interface {}": // repeated object
			// naive implementation of comparison of repeated messages.
			// TODO investigate a more performant way to compare
			items := jsonMap[key].([]interface{})
			otherItems := otherJsonMap[key].([]interface{})
			if len(items) != len(otherItems) {
				return false
			}
			for _, item := range items {
				var found = false
				for _, otherItem := range otherItems {
					itemType := fmt.Sprintf("%T", item)
					otherItemType := fmt.Sprintf("%T", otherItem)
					if itemType != otherItemType {
						// Not sure if they can be different
						continue
					}
					switch itemType {
					case "map[string]interface {}":
						if jsonStringMatches(item.(map[string]interface{}), otherItem.(map[string]interface{}), mustBeEqual) {
							found = true
							break
						}
					default:
						if item == otherItem {
							found = true
							break
						}
					}

				}
				if !found {
					return false
				}
			}
			continue
		}
		if value != otherValue {
			return false
		}
	}
	return true
}

type InvalidStubResponse struct {
	Errors  []string `json:"errors"`
	Example Stub     `json:"example"`
}
