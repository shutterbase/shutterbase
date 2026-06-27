package util

import "github.com/google/uuid"

func BoolPointer(v bool) *bool           { return &v }
func StringPointer(v string) *string     { return &v }
func IntPointer(v int) *int              { return &v }
func UuidPointer(v uuid.UUID) *uuid.UUID { return &v }
