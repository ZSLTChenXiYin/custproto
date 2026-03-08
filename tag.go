package custproto

import (
	"fmt"
	"strings"
)

const (
	tag_required = "required"
)

type tagInformation struct {
	Required  bool
	FieldName string
}

func parseTag(tag_string string) (tagInformation, error) {
	if tag_string == "" {
		return tagInformation{}, fmt.Errorf("tag is empty")
	}

	parts := strings.Split(tag_string, ",")
	if len(parts) > 2 || len(parts) == 0 {
		return tagInformation{}, fmt.Errorf("invalid tag: %s", tag_string)
	}

	var tag_information tagInformation
	is_tag_required := false
	is_tag_field_name := false
	for _, part := range parts {
		if part == "" {
			return tagInformation{}, fmt.Errorf("tag is empty")
		}

		prepart := strings.TrimSpace(part)
		if prepart == tag_required {
			if !is_tag_required {
				is_tag_required = true
				tag_information.Required = true
			} else {
				return tagInformation{}, fmt.Errorf("tag %s is repeated", tag_required)
			}
		} else {
			if !is_tag_field_name {
				is_tag_field_name = true
				tag_information.FieldName = prepart
			} else {
				return tagInformation{}, fmt.Errorf("tag field name is repeated")
			}
		}
	}

	return tag_information, nil
}
