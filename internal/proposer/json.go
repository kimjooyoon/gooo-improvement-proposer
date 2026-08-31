package proposer

import "encoding/json"

func jsonMarshal(value any) ([]byte, error) {
	return json.Marshal(value)
}

func jsonMarshalIndent(value any) ([]byte, error) {
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(raw, '\n'), nil
}
