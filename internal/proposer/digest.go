package proposer

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
)

var commitPattern = regexp.MustCompile(`^[0-9a-f]{40}$`)

func DigestBytes(data []byte) string {
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func ValidateDigest(value string) error {
	if len(value) != len("sha256:")+64 || value[:len("sha256:")] != "sha256:" {
		return errors.New("digest must use sha256:<64 lowercase hex> format")
	}
	if _, err := hex.DecodeString(value[len("sha256:"):]); err != nil {
		return fmt.Errorf("invalid digest %q: %w", value, err)
	}
	return nil
}

func validateCommit(value string) error {
	if !commitPattern.MatchString(value) {
		return fmt.Errorf("commit_sha must use 40 lowercase hexadecimal characters")
	}
	return nil
}

func DigestEvent(event CandidateEvent) string {
	event.EventDigest = ""
	data, err := marshalJSON(event)
	if err != nil {
		return DigestBytes([]byte(fmt.Sprintf("%v", event)))
	}
	return DigestBytes(data)
}

func marshalJSON(value any) ([]byte, error) {
	data, err := jsonMarshal(value)
	if err != nil {
		return nil, err
	}
	return append(data, '\n'), nil
}
