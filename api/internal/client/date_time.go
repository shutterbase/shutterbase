package client

import "time"

type DateTime struct {
	time.Time
}

const customTimeFormat = "2006-01-02 15:04:05.000Z"

func (dt *DateTime) UnmarshalJSON(data []byte) error {
	str := string(data[1 : len(data)-1])
	parsedTime, err := time.Parse(customTimeFormat, str)
	if err != nil {
		return err
	}
	*dt = DateTime{parsedTime}
	return nil
}
